package blockarchive

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadEntries(t *testing.T) {
	buf := bytes.Join([][]byte{
		e2Entry(typeVersion, nil),
		e2Entry(typeCompressedSignedBeaconBlock, []byte{1, 2, 3}),
		e2Entry(0xbeef, []byte("unknown")),
		e2Entry(typeCompressedSignedBeaconBlock, []byte{4}),
	}, nil)

	entries, err := readEntries(buf, 100)
	require.NoError(t, err)
	require.Len(t, entries, 4)

	assert.Equal(t, entry{typ: typeVersion, offset: 100, data: []byte{}}, entries[0])
	assert.Equal(t, entry{typ: typeCompressedSignedBeaconBlock, offset: 108, data: []byte{1, 2, 3}}, entries[1])
	assert.Equal(t, uint16(0xbeef), entries[2].typ)
	assert.Equal(t, int64(119), entries[2].offset)
	assert.Equal(t, int64(134), entries[3].offset)
	assert.Equal(t, []byte{4}, entries[3].data)

	empty, err := readEntries(nil, 0)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestReadEntriesRejectsMalformed(t *testing.T) {
	reserved := e2Entry(typeVersion, nil)
	reserved[6] = 1

	tests := map[string][]byte{
		"truncated header": e2Entry(typeVersion, nil)[:5],
		"overrun":          e2Entry(typeCompressedSignedBeaconBlock, []byte{1, 2, 3})[:10],
		"reserved field":   reserved,
	}

	for name, buf := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := readEntries(buf, 0)
			require.Error(t, err)
		})
	}
}

func TestParseSlotIndex(t *testing.T) {
	rec := e2SlotIndex(64, []int64{-100, 0, -40, -1000})

	idx, err := parseSlotIndex(rec, 1000)
	require.NoError(t, err)
	assert.Equal(t, uint64(64), idx.start)
	assert.Equal(t, []int64{-100, 0, -40, -1000}, idx.offsets)
	assert.Equal(t, slotIndexSize(4), int64(len(rec)))

	pos, ok := idx.entryAt(64)
	assert.True(t, ok)
	assert.Equal(t, int64(900), pos)

	_, ok = idx.entryAt(65)
	assert.False(t, ok, "a zero offset is an empty slot")

	_, ok = idx.entryAt(67)
	assert.False(t, ok, "an offset resolving to the start of the file is an empty slot")

	_, ok = idx.entryAt(68)
	assert.False(t, ok, "slots past the index are not covered")

	_, ok = idx.entryAt(63)
	assert.False(t, ok, "slots before the index are not covered")
}

func TestParseSlotIndexRejectsMalformed(t *testing.T) {
	badCount := e2SlotIndex(0, []int64{-8, -16})
	badCount[len(badCount)-8] = 3

	tests := map[string][]byte{
		"wrong type":     e2Entry(typeVersion, make([]byte, 24)),
		"count mismatch": badCount,
		"short":          e2Entry(typeSlotIndex, make([]byte, 8)),
		"two records":    append(e2SlotIndex(0, nil), e2SlotIndex(0, nil)...),
	}

	for name, rec := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseSlotIndex(rec, 1000)
			require.Error(t, err)
		})
	}
}

func TestSlotIndexValidate(t *testing.T) {
	ok := &slotIndex{start: 0, offsets: []int64{-92, 0, -80, -50}, pos: 100}
	require.NoError(t, ok.validate(8, 60))

	tests := map[string]*slotIndex{
		"out of order":    {offsets: []int64{-50, -80}, pos: 100},
		"duplicate":       {offsets: []int64{-50, -50}, pos: 100},
		"before lo":       {offsets: []int64{-96}, pos: 100},
		"at or after hi":  {offsets: []int64{-40}, pos: 100},
		"negative result": {offsets: []int64{-200}, pos: 100},
	}

	for name, idx := range tests {
		t.Run(name, func(t *testing.T) {
			require.Error(t, idx.validate(8, 60))
		})
	}
}

func TestSlotIndexSpan(t *testing.T) {
	// Slots 10..17 at absolute positions: 10->8, 11 empty, 12->30, 13 empty, 14 empty, 15->70, 16 empty, 17->90; entries end at 120.
	idx := &slotIndex{start: 10, offsets: []int64{-192, 0, -170, 0, -200, -130, 0, -110}, pos: 200}

	tests := []struct {
		name        string
		first, last uint64
		lo, hi      int64
		ok          bool
	}{
		{name: "whole index", first: 10, last: 17, lo: 8, hi: 120, ok: true},
		{name: "ends before next entry", first: 10, last: 11, lo: 8, hi: 30, ok: true},
		{name: "starts on empty slot", first: 11, last: 12, lo: 30, hi: 70, ok: true},
		{name: "skips empty slots to next entry", first: 12, last: 13, lo: 30, hi: 70, ok: true},
		{name: "last window", first: 16, last: 17, lo: 90, hi: 120, ok: true},
		{name: "only empty slots", first: 13, last: 14, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lo, hi, ok := idx.span(tt.first, tt.last, 120)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.lo, lo)
			assert.Equal(t, tt.hi, hi)
		})
	}
}

func TestIndexedEntries(t *testing.T) {
	// Slots 0..3: block, empty, block, block; an unknown entry sits between the first two blocks.
	buf := bytes.Join([][]byte{
		e2Entry(typeCompressedSignedBeaconBlock, []byte("b0")),
		e2Entry(0x4242, []byte("ext")),
		e2Entry(typeCompressedSignedBeaconBlock, []byte("b2")),
		e2Entry(typeCompressedSignedBeaconBlock, []byte("b3")),
	}, nil)
	base := int64(8)
	end := base + int64(len(buf))
	idx := &slotIndex{start: 0, offsets: []int64{base - end, 0, base + 21 - end, base + 31 - end}, pos: end}

	raw, err := indexedEntries(idx, typeCompressedSignedBeaconBlock, 0, 3, buf, base)
	require.NoError(t, err)
	assert.Equal(t, map[uint64][]byte{0: []byte("b0"), 2: []byte("b2"), 3: []byte("b3")}, raw)

	_, err = indexedEntries(idx, typeCompressedSignedBeaconBlock, 0, 2, buf, base)
	require.ErrorContains(t, err, "not in the slot index", "an entry the window's index does not place is rejected")

	_, err = indexedEntries(idx, typeCompressedEnvelope, 0, 3, buf, base)
	require.ErrorContains(t, err, "no entry of type", "an index offset must hold an entry of the expected type")
}

func TestDecompress(t *testing.T) {
	data := bytes.Repeat([]byte("ssz"), 1000)

	out, err := decompress(snappyFramed(t, data))
	require.NoError(t, err)
	assert.Equal(t, data, out)

	_, err = decompress([]byte("not snappy"))
	require.Error(t, err)
}
