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
// index, then one gzip member per canonical block, each Gloas+ block with a
// full payload followed by its envelope's member. Index offsets are relative
// to the end of the index (the data section).
const (
	headerSize    = 16
	formatVersion = 2
	compression   = "gzip"
	// maxIndexSize rejects corrupt headers before allocating; a 1000-slot index is ~200KiB.
	maxIndexSize = 64 << 20

	// PayloadFull marks a Gloas+ block whose payload the canonical chain applied; its envelope is bundled.
	PayloadFull = "full"
	// PayloadEmpty marks a Gloas+ block whose payload the canonical chain did not apply (withheld).
	PayloadEmpty = "empty"
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
	LastRoot    string  `json:"last_root"`
	NextRoot    string  `json:"next_root"`
	Slots       []Entry `json:"slots"`
}

// Entry is one slot of a bundle: either missed, or a canonical block whose
// gzip member sits at Offset (relative to the data section) for Length bytes.
// Gloas+ blocks carry Payload; a full payload's envelope member sits at
// EnvelopeOffset for EnvelopeLength bytes.
//
//nolint:tagliatelle // block-archiver's index format uses snake_case keys.
type Entry struct {
	Slot           uint64 `json:"slot"`
	Missed         bool   `json:"missed"`
	BlockRoot      string `json:"block_root"`
	Fork           string `json:"fork"`
	Offset         int64  `json:"offset"`
	Length         int64  `json:"length"`
	Payload        string `json:"payload,omitempty"`
	EnvelopeOffset int64  `json:"envelope_offset,omitempty"`
	EnvelopeLength int64  `json:"envelope_length,omitempty"`
}

// end is the data-section offset just past the entry's last member.
func (e *Entry) end() int64 {
	if e.Payload == PayloadFull {
		return e.EnvelopeOffset + e.EnvelopeLength
	}

	return e.Offset + e.Length
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

		if e.Missed {
			continue
		}

		if e.Offset < 0 || e.Length <= 0 {
			return fmt.Errorf("bundle index slot %d has invalid byte range", e.Slot)
		}

		switch e.Payload {
		case PayloadFull:
			if e.EnvelopeOffset < e.Offset+e.Length || e.EnvelopeLength <= 0 {
				return fmt.Errorf("bundle index slot %d has invalid envelope byte range", e.Slot)
			}
		case "", PayloadEmpty:
			if e.EnvelopeLength != 0 {
				return fmt.Errorf("bundle index slot %d has an envelope without a full payload", e.Slot)
			}
		default:
			return fmt.Errorf("bundle index slot %d has unknown payload %q", e.Slot, e.Payload)
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

func gunzip(member []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(member))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(zr)
}

// decodeEntry decodes an entry's block and envelope members, checking them against the index and each other.
func decodeEntry(ds *dynssz.DynSsz, e *Entry, block, envelope []byte) (Slot, error) {
	ssz, err := gunzip(block)
	if err != nil {
		return Slot{}, fmt.Errorf("slot %d: failed to gunzip block: %w", e.Slot, err)
	}

	decoded, message, err := unmarshalBlock(ds, e.Fork, ssz)
	if err != nil {
		return Slot{}, fmt.Errorf("slot %d: %w", e.Slot, err)
	}

	slot, err := decoded.Slot()
	if err != nil {
		return Slot{}, fmt.Errorf("slot %d: %w", e.Slot, err)
	}

	if uint64(slot) != e.Slot {
		return Slot{}, fmt.Errorf("slot %d: block decodes to slot %d", e.Slot, slot)
	}

	root, err := ds.HashTreeRoot(message)
	if err != nil {
		return Slot{}, fmt.Errorf("slot %d: failed to hash block: %w", e.Slot, err)
	}

	if got := fmt.Sprintf("%#x", root); got != e.BlockRoot {
		return Slot{}, fmt.Errorf("slot %d: block root %s does not match index root %s", e.Slot, got, e.BlockRoot)
	}

	if hasEnvelopes(decoded.Version) != (e.Payload != "") {
		return Slot{}, fmt.Errorf("slot %d: %s block with payload %q", e.Slot, decoded.Version, e.Payload)
	}

	out := Slot{Block: decoded}
	if e.Payload != PayloadFull {
		return out, nil
	}

	out.Envelope, err = decodeEnvelope(ds, decoded, root, envelope)
	if err != nil {
		return Slot{}, fmt.Errorf("slot %d: %w", e.Slot, err)
	}

	return out, nil
}

func hasEnvelopes(v spec.DataVersion) bool {
	return v >= spec.DataVersionGloas
}

// decodeEnvelope decodes a block's envelope and checks it commits to the block and its bid.
func decodeEnvelope(ds *dynssz.DynSsz, block *spec.VersionedSignedBeaconBlock, root phase0.Root, member []byte) (*gloas.SignedExecutionPayloadEnvelope, error) {
	ssz, err := gunzip(member)
	if err != nil {
		return nil, fmt.Errorf("failed to gunzip envelope: %w", err)
	}

	envelope := &gloas.SignedExecutionPayloadEnvelope{}
	if err = ds.UnmarshalSSZ(envelope, ssz); err != nil {
		return nil, fmt.Errorf("failed to decode envelope: %w", err)
	}

	if envelope.Message == nil || envelope.Message.Payload == nil {
		return nil, fmt.Errorf("envelope has no payload")
	}

	if envelope.Message.BeaconBlockRoot != root {
		return nil, fmt.Errorf("envelope commits to block %#x", envelope.Message.BeaconBlockRoot)
	}

	_, hash, err := bidHashes(block)
	if err != nil {
		return nil, err
	}

	if envelope.Message.Payload.BlockHash != hash {
		return nil, fmt.Errorf("envelope payload hash %#x does not match bid block hash %#x",
			envelope.Message.Payload.BlockHash, hash)
	}

	return envelope, nil
}

// bidHashes returns a Gloas+ block's bid parent block hash and promised block hash.
func bidHashes(block *spec.VersionedSignedBeaconBlock) (parent, hash phase0.Hash32, err error) {
	bid, err := block.SignedExecutionPayloadBid()
	if err != nil {
		return parent, hash, err
	}

	switch {
	case bid.Gloas != nil && bid.Gloas.Message != nil:
		return bid.Gloas.Message.ParentBlockHash, bid.Gloas.Message.BlockHash, nil
	case bid.Heze != nil && bid.Heze.Message != nil:
		return bid.Heze.Message.ParentBlockHash, bid.Heze.Message.BlockHash, nil
	default:
		return parent, hash, fmt.Errorf("%s block has no execution payload bid", block.Version)
	}
}

// checkPayloads checks payload statuses: the next bid builds on a full payload's hash, else on its parent hash.
func checkPayloads(entries []Entry, slots Slots) error {
	var (
		prev                 *Entry
		prevParent, prevHash phase0.Hash32
	)

	for i := range entries {
		e := &entries[i]
		if e.Missed || e.Payload == "" {
			continue
		}

		parent, hash, err := bidHashes(slots[phase0.Slot(e.Slot)].Block)
		if err != nil {
			return fmt.Errorf("slot %d: %w", e.Slot, err)
		}

		if prev != nil {
			want := prevParent
			if prev.Payload == PayloadFull {
				want = prevHash
			}

			if parent != want {
				return fmt.Errorf("slot %d bid builds on %#x, which contradicts slot %d payload %q",
					e.Slot, parent, prev.Slot, prev.Payload)
			}
		}

		prev, prevParent, prevHash = e, parent, hash
	}

	return nil
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
