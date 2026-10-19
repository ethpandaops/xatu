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
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/holiman/uint256"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBlock is a canonical block placed in a test bundle, with its envelope when its payload is full.
type testBlock struct {
	slot     uint64
	fork     string
	ssz      []byte
	root     string
	block    any
	payload  string
	envelope []byte
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

// gloasBlock builds a Gloas block bidding testHash(slot) on parentHash, with its envelope when full.
func gloasBlock(t *testing.T, slot uint64, parentHash phase0.Hash32, full bool) testBlock {
	t.Helper()

	msg := &gloas.BeaconBlock{
		Slot:          phase0.Slot(slot),
		ProposerIndex: phase0.ValidatorIndex(slot % 83),
		ParentRoot:    phase0.Root{byte(slot), 2},
		Body: &gloas.BeaconBlockBody{
			ETH1Data:      &phase0.ETH1Data{BlockHash: make([]byte, 32)},
			Graffiti:      [32]byte{'g', 'l', byte(slot)},
			SyncAggregate: &altair.SyncAggregate{SyncCommitteeBits: make([]byte, 64)},
			SignedExecutionPayloadBid: &gloas.SignedExecutionPayloadBid{Message: &gloas.ExecutionPayloadBid{
				ParentBlockHash: parentHash,
				BlockHash:       testHash(slot),
				Slot:            phase0.Slot(slot),
			}},
			ParentExecutionRequests: &gloas.ExecutionRequests{},
		},
	}

	b := encodeTestBlock(t, "gloas", slot, &gloas.SignedBeaconBlock{Message: msg}, msg)
	b.payload = PayloadEmpty

	if full {
		root, err := dynssz.GetGlobalDynSsz().HashTreeRoot(msg)
		require.NoError(t, err)

		b.payload = PayloadFull
		b.envelope = testEnvelope(t, root, testHash(slot))
	}

	return b
}

func testHash(slot uint64) phase0.Hash32 {
	return phase0.Hash32{0xee, byte(slot >> 8), byte(slot)}
}

func testEnvelope(t *testing.T, root phase0.Root, hash phase0.Hash32) []byte {
	t.Helper()

	envelope := &gloas.SignedExecutionPayloadEnvelope{Message: &gloas.ExecutionPayloadEnvelope{
		Payload: &gloas.ExecutionPayload{
			BlockHash:     hash,
			BaseFeePerGas: uint256.NewInt(7),
			Transactions:  []bellatrix.Transaction{{0x01, 0x02}},
		},
		ExecutionRequests: &gloas.ExecutionRequests{},
		BuilderIndex:      4,
		BeaconBlockRoot:   root,
	}}

	ssz, err := dynssz.GetGlobalDynSsz().MarshalSSZ(envelope)
	require.NoError(t, err)

	return ssz
}

// gloasChain builds Gloas blocks whose bids chain the way the canonical chain applies or skips payloads.
func gloasChain(t *testing.T, slots []uint64, withheld map[uint64]bool) []testBlock {
	t.Helper()

	head := phase0.Hash32{0xaa}
	out := make([]testBlock, 0, len(slots))

	for _, slot := range slots {
		b := gloasBlock(t, slot, head, !withheld[slot])
		if !withheld[slot] {
			head = testHash(slot)
		}

		out = append(out, b)
	}

	return out
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

		member := gzipped(t, b.ssz)
		e := Entry{
			Slot: slot, BlockRoot: b.root, Fork: b.fork, Payload: b.payload,
			Offset: int64(data.Len()), Length: int64(len(member)),
		}
		data.Write(member)

		if b.envelope != nil {
			member = gzipped(t, b.envelope)
			e.EnvelopeOffset, e.EnvelopeLength = int64(data.Len()), int64(len(member))
			data.Write(member)
		}

		idx.Slots = append(idx.Slots, e)
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

func gzipped(t *testing.T, raw []byte) []byte {
	t.Helper()

	var member bytes.Buffer

	zw := gzip.NewWriter(&member)
	_, err := zw.Write(raw)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return member.Bytes()
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
	assert.Nil(t, decoded[102].Block)
	assert.Nil(t, decoded[104].Block)

	assert.Equal(t, spec.DataVersionPhase0, decoded[100].Block.Version)
	assert.Equal(t, spec.DataVersionAltair, decoded[103].Block.Version)

	for _, b := range blocks {
		got := decoded[phase0.Slot(b.slot)]
		require.NotNil(t, got.Block)
		assert.Nil(t, got.Envelope)

		root, rerr := got.Block.Root()
		require.NoError(t, rerr)
		assert.Equal(t, b.root, fmt.Sprintf("%#x", root))
	}

	graffiti, err := decoded[103].Block.Graffiti()
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
	assert.NotNil(t, decoded[11].Block)
	assert.NotNil(t, decoded[12].Block)
	assert.Nil(t, decoded[13].Block)

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
		"bad version":   {withUint32(8, 1), "format version 1"},
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

	idx.Slots = []Entry{{Slot: 0, Length: 5, Payload: PayloadFull, EnvelopeOffset: 2, EnvelopeLength: 3}}
	require.ErrorContains(t, idx.validate(), "invalid envelope byte range")

	idx.Slots = []Entry{{Slot: 0, Length: 5, Payload: PayloadEmpty, EnvelopeOffset: 5, EnvelopeLength: 3}}
	require.ErrorContains(t, idx.validate(), "envelope without a full payload")

	idx.Slots = []Entry{{Slot: 0, Length: 5, Payload: "maybe"}}
	require.ErrorContains(t, idx.validate(), "unknown payload")
}

func TestDecodeGloasEntriesWithEnvelopes(t *testing.T) {
	blocks := gloasChain(t, []uint64{200, 201, 203, 204}, map[uint64]bool{201: true, 204: true})
	buf := encodeBundle(t, 200, 205, blocks)

	idx, dataOffset, err := readIndex(bytes.NewReader(buf))
	require.NoError(t, err)

	decoded, err := decodeEntries(dynssz.GetGlobalDynSsz(), idx.Slots, buf[dataOffset:], 0)
	require.NoError(t, err)
	require.Len(t, decoded, 6)

	for _, slot := range []phase0.Slot{200, 203} {
		require.NotNil(t, decoded[slot].Envelope, "slot %d", slot)
		assert.Equal(t, testHash(uint64(slot)), decoded[slot].Envelope.Message.Payload.BlockHash)

		root, rerr := decoded[slot].Block.Root()
		require.NoError(t, rerr)
		assert.Equal(t, root, decoded[slot].Envelope.Message.BeaconBlockRoot)
	}

	for _, slot := range []phase0.Slot{201, 204} {
		assert.NotNil(t, decoded[slot].Block)
		assert.Nil(t, decoded[slot].Envelope, "withheld slot %d", slot)
	}

	assert.Nil(t, decoded[202].Block)

	// A window that starts at a withheld block and ends in a missed slot.
	window := idx.entries(201, 202)
	lo, hi := window[0].Offset, window[0].end()
	decoded, err = decodeEntries(dynssz.GetGlobalDynSsz(), window, buf[dataOffset+lo:dataOffset+hi], lo)
	require.NoError(t, err)
	assert.NotNil(t, decoded[201].Block)
}

func TestDecodeRejectsInconsistentPayloads(t *testing.T) {
	ds := dynssz.GetGlobalDynSsz()

	// The index claims 201 was withheld but 203's bid builds on 201's promised hash.
	blocks := gloasChain(t, []uint64{200, 201, 203}, nil)
	blocks[1].payload, blocks[1].envelope = PayloadEmpty, nil
	buf := encodeBundle(t, 200, 203, blocks)
	idx, dataOffset, err := readIndex(bytes.NewReader(buf))
	require.NoError(t, err)

	_, err = decodeEntries(ds, idx.Slots, buf[dataOffset:], 0)
	require.ErrorContains(t, err, "contradicts slot 201")

	// An envelope for another block.
	blocks = gloasChain(t, []uint64{300, 301}, nil)
	blocks[0].envelope = blocks[1].envelope
	buf = encodeBundle(t, 300, 301, blocks)
	idx, dataOffset, err = readIndex(bytes.NewReader(buf))
	require.NoError(t, err)

	_, err = decodeEntries(ds, idx.Slots, buf[dataOffset:], 0)
	require.ErrorContains(t, err, "envelope commits to block")

	// A Gloas block without a payload status, and a pre-Gloas block with one.
	blocks = gloasChain(t, []uint64{400}, nil)
	blocks[0].payload, blocks[0].envelope = "", nil
	buf = encodeBundle(t, 400, 400, blocks)
	idx, dataOffset, err = readIndex(bytes.NewReader(buf))
	require.NoError(t, err)

	_, err = decodeEntries(ds, idx.Slots, buf[dataOffset:], 0)
	require.ErrorContains(t, err, "with payload")

	p0 := phase0Block(t, 500)
	p0.payload = PayloadEmpty
	buf = encodeBundle(t, 500, 500, []testBlock{p0})
	idx, dataOffset, err = readIndex(bytes.NewReader(buf))
	require.NoError(t, err)

	_, err = decodeEntries(ds, idx.Slots, buf[dataOffset:], 0)
	require.ErrorContains(t, err, "with payload")
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
	_, err = decodeEntry(ds, &e, member, nil)
	require.ErrorContains(t, err, "does not match index root")

	e = idx.Slots[0]
	e.Slot = 6
	_, err = decodeEntry(ds, &e, member, nil)
	require.ErrorContains(t, err, "decodes to slot 5")

	e = idx.Slots[0]
	e.Fork = "nonsense"
	_, err = decodeEntry(ds, &e, member, nil)
	require.Error(t, err)

	e = idx.Slots[0]
	_, err = decodeEntry(ds, &e, []byte("not gzip"), nil)
	require.Error(t, err)
}
