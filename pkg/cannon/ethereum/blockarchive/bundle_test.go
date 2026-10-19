package blockarchive

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBlock is a canonical block placed in a test bundle.
type testBlock struct {
	slot  uint64
	fork  string
	ssz   []byte
	root  string
	block any
}

func phase0Block(t *testing.T, slot uint64) testBlock {
	t.Helper()

	msg := &phase0.BeaconBlock{
		Slot:          phase0.Slot(slot),
		ProposerIndex: phase0.ValidatorIndex(slot % 97),
		ParentRoot:    phase0.Root{byte(slot)},
		Body: &phase0.BeaconBlockBody{
			ETH1Data: &phase0.ETH1Data{BlockHash: make([]byte, 32)},
			Graffiti: [32]byte{'p', '0', byte(slot)},
		},
	}

	return encodeTestBlock(t, "phase0", slot, &phase0.SignedBeaconBlock{Message: msg}, msg)
}

func altairBlock(t *testing.T, slot uint64) testBlock {
	t.Helper()

	msg := &altair.BeaconBlock{
		Slot:          phase0.Slot(slot),
		ProposerIndex: phase0.ValidatorIndex(slot % 89),
		ParentRoot:    phase0.Root{byte(slot), 1},
		Body: &altair.BeaconBlockBody{
			ETH1Data:      &phase0.ETH1Data{BlockHash: make([]byte, 32)},
			Graffiti:      [32]byte{'a', 'l', byte(slot)},
			SyncAggregate: &altair.SyncAggregate{SyncCommitteeBits: make([]byte, 64)},
		},
	}

	return encodeTestBlock(t, "altair", slot, &altair.SignedBeaconBlock{Message: msg}, msg)
}

func encodeTestBlock(t *testing.T, fork string, slot uint64, signed, msg any) testBlock {
	t.Helper()

	ds := dynssz.GetGlobalDynSsz()

	ssz, err := ds.MarshalSSZ(signed)
	require.NoError(t, err)

	root, err := ds.HashTreeRoot(msg)
	require.NoError(t, err)

	return testBlock{slot: slot, fork: fork, ssz: ssz, root: fmt.Sprintf("%#x", root), block: signed}
}

// encodeBundle writes the block-archiver bundle format for start..end.
func encodeBundle(t *testing.T, start, end uint64, blocks []testBlock) []byte {
	t.Helper()

	bySlot := make(map[uint64]testBlock, len(blocks))
	for _, b := range blocks {
		bySlot[b.slot] = b
	}

	idx := Index{Version: formatVersion, Network: testNetwork, StartSlot: start, EndSlot: end, Compression: compression}

	var data bytes.Buffer

	for slot := start; slot <= end; slot++ {
		b, ok := bySlot[slot]
		if !ok {
			idx.Slots = append(idx.Slots, Entry{Slot: slot, Missed: true})

			continue
		}

		var member bytes.Buffer

		zw := gzip.NewWriter(&member)
		_, err := zw.Write(b.ssz)
		require.NoError(t, err)
		require.NoError(t, zw.Close())

		idx.Slots = append(idx.Slots, Entry{
			Slot: slot, BlockRoot: b.root, Fork: b.fork,
			Offset: int64(data.Len()), Length: int64(member.Len()),
		})
		data.Write(member.Bytes())
	}

	raw, err := json.Marshal(idx)
	require.NoError(t, err)

	out := make([]byte, headerSize, headerSize+len(raw)+data.Len())
	copy(out, magic)
	binary.LittleEndian.PutUint32(out[8:12], formatVersion)
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(raw)))
	out = append(out, raw...)

	return append(out, data.Bytes()...)
}

func TestReadIndexAndDecode(t *testing.T) {
	blocks := []testBlock{phase0Block(t, 100), phase0Block(t, 101), altairBlock(t, 103)}
	buf := encodeBundle(t, 100, 104, blocks)

	idx, dataOffset, err := readIndex(bytes.NewReader(buf))
	require.NoError(t, err)
	assert.Equal(t, uint64(100), idx.StartSlot)
	assert.Equal(t, uint64(104), idx.EndSlot)
	assert.Len(t, idx.entries(100, 104), 5)
	assert.Len(t, idx.entries(103, 200), 2)
	assert.Empty(t, idx.entries(200, 300))

	decoded, err := decodeEntries(dynssz.GetGlobalDynSsz(), idx.entries(100, 104), buf[dataOffset:], 0)
	require.NoError(t, err)
	require.Len(t, decoded, 5)
	assert.Nil(t, decoded[102])
	assert.Nil(t, decoded[104])

	assert.Equal(t, spec.DataVersionPhase0, decoded[100].Version)
	assert.Equal(t, spec.DataVersionAltair, decoded[103].Version)

	for _, b := range blocks {
		got := decoded[phase0.Slot(b.slot)]
		require.NotNil(t, got)

		root, rerr := got.Root()
		require.NoError(t, rerr)
		assert.Equal(t, b.root, fmt.Sprintf("%#x", root))
	}

	graffiti, err := decoded[103].Graffiti()
	require.NoError(t, err)
	assert.Equal(t, [32]byte{'a', 'l', 103}, graffiti)
}

func TestDecodeEntriesFromWindow(t *testing.T) {
	blocks := []testBlock{phase0Block(t, 10), phase0Block(t, 11), phase0Block(t, 12)}
	buf := encodeBundle(t, 10, 13, blocks)

	idx, dataOffset, err := readIndex(bytes.NewReader(buf))
	require.NoError(t, err)

	window := idx.entries(11, 13)
	lo := window[0].Offset
	hi := window[1].Offset + window[1].Length

	decoded, err := decodeEntries(dynssz.GetGlobalDynSsz(), window, buf[dataOffset+lo:dataOffset+hi], lo)
	require.NoError(t, err)
	require.Len(t, decoded, 3)
	assert.NotNil(t, decoded[11])
	assert.NotNil(t, decoded[12])
	assert.Nil(t, decoded[13])

	_, err = decodeEntries(dynssz.GetGlobalDynSsz(), idx.entries(10, 13), buf[dataOffset+lo:dataOffset+hi], lo)
	require.ErrorContains(t, err, "outside fetched data")
}

func TestReadIndexRejectsCorruptHeaders(t *testing.T) {
	buf := encodeBundle(t, 0, 1, []testBlock{phase0Block(t, 0)})

	withUint32 := func(at int, v uint32) []byte {
		b := bytes.Clone(buf)
		binary.LittleEndian.PutUint32(b[at:at+4], v)

		return b
	}

	cases := map[string]struct {
		input []byte
		err   string
	}{
		"bad magic":     {append([]byte("XXXXXXXX"), buf[8:]...), "not a block bundle"},
		"bad version":   {withUint32(8, 9), "format version 9"},
		"truncated":     {buf[:10], "header"},
		"index too big": {withUint32(12, 1<<30), "exceeds"},
		"short index":   {buf[:headerSize+5], "index"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := readIndex(bytes.NewReader(tc.input))
			require.ErrorContains(t, err, tc.err)
		})
	}
}

func TestIndexValidate(t *testing.T) {
	idx := &Index{Compression: "zstd", StartSlot: 0, EndSlot: 0, Slots: []Entry{{Slot: 0, Missed: true}}}
	require.ErrorContains(t, idx.validate(), "compression")

	idx = &Index{Compression: compression, StartSlot: 0, EndSlot: 1, Slots: []Entry{{Slot: 0, Missed: true}}}
	require.ErrorContains(t, idx.validate(), "has 1 slots")

	idx = &Index{Compression: compression, StartSlot: 0, EndSlot: 1, Slots: []Entry{{Slot: 0, Missed: true}, {Slot: 2, Missed: true}}}
	require.ErrorContains(t, idx.validate(), "want 1")

	idx = &Index{Compression: compression, StartSlot: 0, EndSlot: 0, Slots: []Entry{{Slot: 0, Length: 0}}}
	require.ErrorContains(t, idx.validate(), "invalid byte range")
}

func TestDecodeEntryChecksIndex(t *testing.T) {
	ds := dynssz.GetGlobalDynSsz()
	b := phase0Block(t, 5)
	buf := encodeBundle(t, 5, 5, []testBlock{b})

	idx, dataOffset, err := readIndex(bytes.NewReader(buf))
	require.NoError(t, err)

	member := buf[dataOffset:]

	e := idx.Slots[0]
	e.BlockRoot = "0x" + fmt.Sprintf("%064x", 1)
	_, err = decodeEntry(ds, &e, member)
	require.ErrorContains(t, err, "does not match index root")

	e = idx.Slots[0]
	e.Slot = 6
	_, err = decodeEntry(ds, &e, member)
	require.ErrorContains(t, err, "decodes to slot 5")

	e = idx.Slots[0]
	e.Fork = "nonsense"
	_, err = decodeEntry(ds, &e, member)
	require.Error(t, err)

	e = idx.Slots[0]
	_, err = decodeEntry(ds, &e, []byte("not gzip"))
	require.Error(t, err)
}
