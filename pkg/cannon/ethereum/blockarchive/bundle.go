package blockarchive

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/heze"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/go-eth2-client/spec/version"
	dynssz "github.com/pk910/dynamic-ssz"
)

// A bundle is a 16-byte header (magic, format version, index length), a JSON
// index, then one gzip member per canonical block. Index offsets are relative
// to the end of the index (the data section).
const (
	headerSize    = 16
	formatVersion = 1
	compression   = "gzip"
	// maxIndexSize rejects corrupt headers before allocating; a 1000-slot index is ~150KiB.
	maxIndexSize = 64 << 20
)

var magic = []byte("EPOBNDL\x00")

// Index is a bundle's table of contents: one entry per slot of its range.
//
//nolint:tagliatelle // block-archiver's index format uses snake_case keys.
type Index struct {
	Version     int     `json:"version"`
	Network     string  `json:"network"`
	StartSlot   uint64  `json:"start_slot"`
	EndSlot     uint64  `json:"end_slot"`
	Compression string  `json:"compression"`
	ParentRoot  string  `json:"parent_root"`
	Slots       []Entry `json:"slots"`
}

// Entry is one slot of a bundle: either missed, or a canonical block whose
// gzip member sits at Offset (relative to the data section) for Length bytes.
//
//nolint:tagliatelle // block-archiver's index format uses snake_case keys.
type Entry struct {
	Slot      uint64 `json:"slot"`
	Missed    bool   `json:"missed"`
	BlockRoot string `json:"block_root"`
	Fork      string `json:"fork"`
	Offset    int64  `json:"offset"`
	Length    int64  `json:"length"`
}

// indexLength validates a bundle header and returns the length of the index after it.
func indexLength(header []byte) (int64, error) {
	if len(header) < headerSize {
		return 0, fmt.Errorf("bundle header truncated: %d bytes", len(header))
	}

	if !bytes.Equal(header[:len(magic)], magic) {
		return 0, fmt.Errorf("not a block bundle")
	}

	if v := binary.LittleEndian.Uint32(header[8:12]); v != formatVersion {
		return 0, fmt.Errorf("unsupported bundle format version %d", v)
	}

	n := int64(binary.LittleEndian.Uint32(header[12:16]))
	if n > maxIndexSize {
		return 0, fmt.Errorf("bundle index length %d exceeds %d", n, maxIndexSize)
	}

	return n, nil
}

// readIndex reads a bundle's header and index from r, returning the index and
// the absolute offset of the data section.
func readIndex(r io.Reader) (*Index, int64, error) {
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, 0, fmt.Errorf("failed to read bundle header: %w", err)
	}

	n, err := indexLength(header)
	if err != nil {
		return nil, 0, err
	}

	raw := make([]byte, n)
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, 0, fmt.Errorf("failed to read bundle index: %w", err)
	}

	idx := &Index{}
	if err := json.Unmarshal(raw, idx); err != nil {
		return nil, 0, fmt.Errorf("failed to decode bundle index: %w", err)
	}

	if err := idx.validate(); err != nil {
		return nil, 0, err
	}

	return idx, headerSize + n, nil
}

func (idx *Index) validate() error {
	if idx.Compression != compression {
		return fmt.Errorf("unsupported bundle compression %q", idx.Compression)
	}

	if idx.EndSlot < idx.StartSlot || uint64(len(idx.Slots)) != idx.EndSlot-idx.StartSlot+1 {
		return fmt.Errorf("bundle index for %d..%d has %d slots", idx.StartSlot, idx.EndSlot, len(idx.Slots))
	}

	for i, e := range idx.Slots {
		if e.Slot != idx.StartSlot+uint64(i) {
			return fmt.Errorf("bundle index entry %d is slot %d, want %d", i, e.Slot, idx.StartSlot+uint64(i))
		}

		if !e.Missed && (e.Offset < 0 || e.Length <= 0) {
			return fmt.Errorf("bundle index slot %d has invalid byte range", e.Slot)
		}
	}

	return nil
}

// entries returns the index entries for slots first..last inclusive.
func (idx *Index) entries(first, last uint64) []Entry {
	if first < idx.StartSlot {
		first = idx.StartSlot
	}

	if last > idx.EndSlot {
		last = idx.EndSlot
	}

	if first > last {
		return nil
	}

	return idx.Slots[first-idx.StartSlot : last-idx.StartSlot+1]
}

// decodeEntry gunzips and SSZ-decodes one bundle entry, checking that the
// block's slot and root match the index.
func decodeEntry(ds *dynssz.DynSsz, e *Entry, member []byte) (*spec.VersionedSignedBeaconBlock, error) {
	zr, err := gzip.NewReader(bytes.NewReader(member))
	if err != nil {
		return nil, fmt.Errorf("slot %d: %w", e.Slot, err)
	}

	ssz, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("slot %d: failed to gunzip block: %w", e.Slot, err)
	}

	block, message, err := unmarshalBlock(ds, e.Fork, ssz)
	if err != nil {
		return nil, fmt.Errorf("slot %d: %w", e.Slot, err)
	}

	slot, err := block.Slot()
	if err != nil {
		return nil, fmt.Errorf("slot %d: %w", e.Slot, err)
	}

	if uint64(slot) != e.Slot {
		return nil, fmt.Errorf("slot %d: block decodes to slot %d", e.Slot, slot)
	}

	root, err := ds.HashTreeRoot(message)
	if err != nil {
		return nil, fmt.Errorf("slot %d: failed to hash block: %w", e.Slot, err)
	}

	if got := fmt.Sprintf("%#x", root); got != e.BlockRoot {
		return nil, fmt.Errorf("slot %d: block root %s does not match index root %s", e.Slot, got, e.BlockRoot)
	}

	return block, nil
}

// unmarshalBlock decodes a SignedBeaconBlock of the named fork the same way
// go-eth2-client decodes an SSZ beacon API response, returning the versioned
// block and its message.
func unmarshalBlock(ds *dynssz.DynSsz, fork string, ssz []byte) (*spec.VersionedSignedBeaconBlock, any, error) {
	v, err := version.DataVersionFromString(fork)
	if err != nil {
		return nil, nil, err
	}

	block := &spec.VersionedSignedBeaconBlock{Version: v}

	var (
		target  any
		message func() any
	)

	switch v {
	case spec.DataVersionPhase0:
		block.Phase0 = &phase0.SignedBeaconBlock{}
		target, message = block.Phase0, func() any { return block.Phase0.Message }
	case spec.DataVersionAltair:
		block.Altair = &altair.SignedBeaconBlock{}
		target, message = block.Altair, func() any { return block.Altair.Message }
	case spec.DataVersionBellatrix:
		block.Bellatrix = &bellatrix.SignedBeaconBlock{}
		target, message = block.Bellatrix, func() any { return block.Bellatrix.Message }
	case spec.DataVersionCapella:
		block.Capella = &capella.SignedBeaconBlock{}
		target, message = block.Capella, func() any { return block.Capella.Message }
	case spec.DataVersionDeneb:
		block.Deneb = &deneb.SignedBeaconBlock{}
		target, message = block.Deneb, func() any { return block.Deneb.Message }
	case spec.DataVersionElectra:
		block.Electra = &electra.SignedBeaconBlock{}
		target, message = block.Electra, func() any { return block.Electra.Message }
	case spec.DataVersionFulu:
		block.Fulu = &electra.SignedBeaconBlock{}
		target, message = block.Fulu, func() any { return block.Fulu.Message }
	case spec.DataVersionGloas:
		block.Gloas = &gloas.SignedBeaconBlock{}
		target, message = block.Gloas, func() any { return block.Gloas.Message }
	case spec.DataVersionHeze:
		block.Heze = &heze.SignedBeaconBlock{}
		target, message = block.Heze, func() any { return block.Heze.Message }
	default:
		return nil, nil, fmt.Errorf("unhandled block fork %q", fork)
	}

	if err := ds.UnmarshalSSZ(target, ssz); err != nil {
		return nil, nil, fmt.Errorf("failed to decode %s block: %w", fork, err)
	}

	return block, message(), nil
}
