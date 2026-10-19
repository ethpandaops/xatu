package blockarchive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/snappy"
)

// e2store entry types, see https://github.com/eth-clients/e2store-format-specs.
const (
	typeVersion                     uint16 = 0x3265
	typeCompressedSignedBeaconBlock uint16 = 0x0001
	typeCompressedBeaconState       uint16 = 0x0002
	typeSlotIndex                   uint16 = 0x3269
	// typeCompressedEnvelope is a snappy-framed SSZ SignedExecutionPayloadEnvelope in block-archiver's envelope sidecar.
	typeCompressedEnvelope uint16 = 0x0180

	entryHeaderSize = 8
	// maxEntrySize bounds a decompressed entry so a corrupt or hostile entry cannot exhaust memory.
	maxEntrySize = 64 << 20
)

// entry is one e2store record; offset is the absolute file position of its header.
type entry struct {
	typ    uint16
	offset int64
	data   []byte
}

// readEntries parses the consecutive e2store records in buf, the file bytes from position base.
func readEntries(buf []byte, base int64) ([]entry, error) {
	var out []entry

	for pos := int64(0); pos < int64(len(buf)); {
		if int64(len(buf))-pos < entryHeaderSize {
			return nil, fmt.Errorf("truncated e2store header at offset %d", base+pos)
		}

		header := buf[pos : pos+entryHeaderSize]
		typ := binary.LittleEndian.Uint16(header[0:2])
		length := int64(binary.LittleEndian.Uint32(header[2:6]))

		if binary.LittleEndian.Uint16(header[6:8]) != 0 {
			return nil, fmt.Errorf("e2store entry at offset %d has a non-zero reserved field", base+pos)
		}

		start := pos + entryHeaderSize
		if length > int64(len(buf))-start {
			return nil, fmt.Errorf("e2store entry at offset %d overruns the data (%d bytes)", base+pos, length)
		}

		out = append(out, entry{typ: typ, offset: base + pos, data: buf[start : start+length]})
		pos = start + length
	}

	return out, nil
}

// slotIndex is an e2store SlotIndex record; offsets are relative to the record's position.
type slotIndex struct {
	start   uint64
	offsets []int64
	pos     int64
}

// slotIndexSize is the encoded size of a SlotIndex record covering count slots.
func slotIndexSize(count uint64) int64 {
	return entryHeaderSize + 16 + 8*int64(count) //nolint:gosec // count is a slots-per-era constant.
}

// parseSlotIndex decodes the SlotIndex record rec (header included) found at absolute position pos.
func parseSlotIndex(rec []byte, pos int64) (*slotIndex, error) {
	entries, err := readEntries(rec, pos)
	if err != nil {
		return nil, err
	}

	if len(entries) != 1 || entries[0].typ != typeSlotIndex {
		return nil, fmt.Errorf("no slot index at offset %d", pos)
	}

	data := entries[0].data
	if len(data) < 16 || len(data)%8 != 0 {
		return nil, fmt.Errorf("slot index at offset %d has invalid length %d", pos, len(data))
	}

	count, offsets := binary.LittleEndian.Uint64(data[len(data)-8:]), len(data)/8-2
	if count != uint64(offsets) { //nolint:gosec // offsets is non-negative: len(data) >= 16.
		return nil, fmt.Errorf("slot index at offset %d has count %d for %d offsets", pos, count, offsets)
	}

	idx := &slotIndex{
		start:   binary.LittleEndian.Uint64(data[0:8]),
		offsets: make([]int64, offsets),
		pos:     pos,
	}

	for i := range idx.offsets {
		idx.offsets[i] = int64(binary.LittleEndian.Uint64(data[8+8*i:])) //nolint:gosec // offsets are signed by spec.
	}

	return idx, nil
}

func (s *slotIndex) covers(slot uint64) bool {
	return slot >= s.start && slot-s.start < uint64(len(s.offsets))
}

// entryAt returns the absolute position of slot's entry; ok is false when the slot has none.
func (s *slotIndex) entryAt(slot uint64) (pos int64, ok bool) {
	if !s.covers(slot) {
		return 0, false
	}

	off := s.offsets[slot-s.start]
	pos = s.pos + off

	// Writers mark an empty slot with a zero offset or, as Nimbus does, one resolving to the start of the file.
	if off == 0 || pos == 0 {
		return 0, false
	}

	return pos, true
}

// validate checks that the indexed entries lie in [lo, hi) in strictly increasing slot order.
func (s *slotIndex) validate(lo, hi int64) error {
	prev := lo - 1

	for i := range s.offsets {
		slot := s.start + uint64(i)

		pos, ok := s.entryAt(slot)
		if !ok {
			continue
		}

		if pos <= prev || pos >= hi {
			return fmt.Errorf("slot %d entry offset %d is out of order or outside [%d, %d)", slot, pos, lo, hi)
		}

		prev = pos
	}

	return nil
}

// span returns the byte range [lo, hi) of slots first..last's entries; ok is false when they have none.
func (s *slotIndex) span(first, last uint64, end int64) (lo, hi int64, ok bool) {
	for slot := first; slot <= last; slot++ {
		if pos, has := s.entryAt(slot); has {
			lo, ok = pos, true

			break
		}
	}

	if !ok {
		return 0, 0, false
	}

	hi = end

	for slot := last + 1; s.covers(slot); slot++ {
		if pos, has := s.entryAt(slot); has {
			hi = pos

			break
		}
	}

	return lo, hi, true
}

// indexedEntries maps slots first..last to their typ entries in buf; every typ entry must be indexed, others are skipped.
func indexedEntries(idx *slotIndex, typ uint16, first, last uint64, buf []byte, base int64) (map[uint64][]byte, error) {
	entries, err := readEntries(buf, base)
	if err != nil {
		return nil, err
	}

	byOffset := make(map[int64][]byte, len(entries))

	for _, e := range entries {
		if e.typ == typ {
			byOffset[e.offset] = e.data
		}
	}

	out := make(map[uint64][]byte, len(byOffset))

	for slot := first; slot <= last; slot++ {
		pos, ok := idx.entryAt(slot)
		if !ok {
			continue
		}

		data, found := byOffset[pos]
		if !found {
			return nil, fmt.Errorf("slot %d: no entry of type %#04x at offset %d", slot, typ, pos)
		}

		out[slot] = data
	}

	if len(out) != len(byOffset) {
		return nil, fmt.Errorf("%d entries of type %#04x are not in the slot index", len(byOffset)-len(out), typ)
	}

	return out, nil
}

// decompress decodes a snappy framed entry.
func decompress(data []byte) ([]byte, error) {
	out, err := io.ReadAll(io.LimitReader(snappy.NewReader(bytes.NewReader(data)), maxEntrySize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress entry: %w", err)
	}

	if len(out) > maxEntrySize {
		return nil, fmt.Errorf("entry decompresses to more than %d bytes", maxEntrySize)
	}

	return out, nil
}
