package blockarchive

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/golang/snappy"
	"github.com/holiman/uint256"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/require"
)

const (
	testNetwork       = "testnet"
	testSlotsPerEra   = 16
	testSlotsPerEpoch = 4
	// testAltairEpoch puts the phase0 to altair boundary in the middle of era 2 (slots 16..31).
	testAltairEpoch = 6
	// testGloasEpoch puts the altair to gloas boundary in the middle of era 3 (slots 32..47).
	testGloasEpoch = 9
)

func testChain() *Chain {
	return &Chain{
		Codec:         dynssz.GetGlobalDynSsz(),
		SlotsPerEpoch: testSlotsPerEpoch,
		SlotsPerEra:   testSlotsPerEra,
		Forks: state.ForkEpochs{
			{Epoch: testAltairEpoch, Name: spec.DataVersionAltair},
			{Epoch: testGloasEpoch, Name: spec.DataVersionGloas},
			{Epoch: 1 << 40, Name: spec.DataVersionHeze},
		},
	}
}

// testBlock is a canonical block placed in a test era, with its envelope when its payload was applied.
type testBlock struct {
	slot     uint64
	ssz      []byte
	envelope []byte
}

// testBlockAt builds the block the test chain's fork schedule expects at slot.
func testBlockAt(t *testing.T, slot uint64) testBlock {
	t.Helper()

	switch testChain().forkAt(slot) {
	case spec.DataVersionPhase0:
		return phase0Block(t, slot)
	case spec.DataVersionAltair:
		return altairBlock(t, slot)
	default:
		return gloasBlock(t, slot, phase0.Hash32{0xaa}, false)
	}
}

func testBlocksAt(t *testing.T, slots ...uint64) []testBlock {
	t.Helper()

	out := make([]testBlock, 0, len(slots))
	for _, slot := range slots {
		out = append(out, testBlockAt(t, slot))
	}

	return out
}

func phase0Block(t *testing.T, slot uint64) testBlock {
	t.Helper()

	return encodeTestBlock(t, slot, &phase0.SignedBeaconBlock{Message: &phase0.BeaconBlock{
		Slot:          phase0.Slot(slot),
		ProposerIndex: phase0.ValidatorIndex(slot % 97),
		ParentRoot:    phase0.Root{byte(slot)},
		Body: &phase0.BeaconBlockBody{
			ETH1Data: &phase0.ETH1Data{BlockHash: make([]byte, 32)},
			Graffiti: [32]byte{'p', '0', byte(slot)},
		},
	}})
}

func altairBlock(t *testing.T, slot uint64) testBlock {
	t.Helper()

	return encodeTestBlock(t, slot, &altair.SignedBeaconBlock{Message: &altair.BeaconBlock{
		Slot:          phase0.Slot(slot),
		ProposerIndex: phase0.ValidatorIndex(slot % 89),
		ParentRoot:    phase0.Root{byte(slot), 1},
		Body: &altair.BeaconBlockBody{
			ETH1Data:      &phase0.ETH1Data{BlockHash: make([]byte, 32)},
			Graffiti:      [32]byte{'a', 'l', byte(slot)},
			SyncAggregate: &altair.SyncAggregate{SyncCommitteeBits: make([]byte, 64)},
		},
	}})
}

// gloasBlock builds a Gloas block bidding testHash(slot) on parentHash, with its envelope when applied.
func gloasBlock(t *testing.T, slot uint64, parentHash phase0.Hash32, applied bool) testBlock {
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

	b := encodeTestBlock(t, slot, &gloas.SignedBeaconBlock{Message: msg})

	if applied {
		root, err := dynssz.GetGlobalDynSsz().HashTreeRoot(msg)
		require.NoError(t, err)

		b.envelope = testEnvelope(t, root, testHash(slot))
	}

	return b
}

func testHash(slot uint64) phase0.Hash32 {
	return phase0.Hash32{0xee, byte(slot >> 8), byte(slot)}
}

func testEnvelope(t *testing.T, root phase0.Root, hash phase0.Hash32) []byte {
	t.Helper()

	ssz, err := dynssz.GetGlobalDynSsz().MarshalSSZ(&gloas.SignedExecutionPayloadEnvelope{Message: &gloas.ExecutionPayloadEnvelope{
		Payload: &gloas.ExecutionPayload{
			BlockHash:     hash,
			BaseFeePerGas: uint256.NewInt(7),
			Transactions:  []bellatrix.Transaction{{0x01, 0x02}},
		},
		ExecutionRequests: &gloas.ExecutionRequests{},
		BuilderIndex:      4,
		BeaconBlockRoot:   root,
	}})
	require.NoError(t, err)

	return ssz
}

// gloasChain builds Gloas blocks whose bids chain the way the canonical chain applies or withholds payloads.
func gloasChain(t *testing.T, slots []uint64, withheld map[uint64]bool) []testBlock {
	t.Helper()

	head := phase0.Hash32{0xaa}
	out := make([]testBlock, 0, len(slots))

	for _, slot := range slots {
		out = append(out, gloasBlock(t, slot, head, !withheld[slot]))

		if !withheld[slot] {
			head = testHash(slot)
		}
	}

	return out
}

func encodeTestBlock(t *testing.T, slot uint64, signed any) testBlock {
	t.Helper()

	ssz, err := dynssz.GetGlobalDynSsz().MarshalSSZ(signed)
	require.NoError(t, err)

	return testBlock{slot: slot, ssz: ssz}
}

func snappyFramed(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	w := snappy.NewBufferedWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return buf.Bytes()
}

func e2Entry(typ uint16, data []byte) []byte {
	out := make([]byte, entryHeaderSize, entryHeaderSize+len(data))
	binary.LittleEndian.PutUint16(out[0:2], typ)
	binary.LittleEndian.PutUint32(out[2:6], uint32(len(data)))

	return append(out, data...)
}

func e2SlotIndex(start uint64, offsets []int64) []byte {
	data := make([]byte, 0, 16+8*len(offsets))
	data = binary.LittleEndian.AppendUint64(data, start)

	for _, o := range offsets {
		data = binary.LittleEndian.AppendUint64(data, uint64(o))
	}

	data = binary.LittleEndian.AppendUint64(data, uint64(len(offsets)))

	return e2Entry(typeSlotIndex, data)
}

// eraOptions shapes an encoded era file.
type eraOptions struct {
	// nimbusMissed marks missed slots the way Nimbus does: an offset resolving to the start of the file.
	nimbusMissed bool
	// unknownEntries inserts an entry of an unknown type after every block.
	unknownEntries bool
}

// encodeEra writes a single-group era file for era holding blocks.
func encodeEra(t *testing.T, era uint64, blocks []testBlock, opts eraOptions) []byte {
	t.Helper()

	start := (era - 1) * testSlotsPerEra
	file := e2Entry(typeVersion, nil)
	positions := make(map[uint64]int64, len(blocks))

	for _, b := range blocks {
		require.True(t, b.slot >= start && b.slot < start+testSlotsPerEra, "slot %d outside era %d", b.slot, era)

		positions[b.slot] = int64(len(file))
		file = append(file, e2Entry(typeCompressedSignedBeaconBlock, snappyFramed(t, b.ssz))...)

		if opts.unknownEntries {
			file = append(file, e2Entry(0x7777, []byte("future record"))...)
		}
	}

	statePos := int64(len(file))
	file = append(file, e2Entry(typeCompressedBeaconState, snappyFramed(t, []byte("state")))...)

	blockIdxPos := int64(len(file))
	offsets := make([]int64, testSlotsPerEra)

	for i := range offsets {
		pos, ok := positions[start+uint64(i)]

		switch {
		case ok:
			offsets[i] = pos - blockIdxPos
		case opts.nimbusMissed:
			offsets[i] = -blockIdxPos
		}
	}

	file = append(file, e2SlotIndex(start, offsets)...)
	stateIdxPos := int64(len(file))

	return append(file, e2SlotIndex(era*testSlotsPerEra, []int64{statePos - stateIdxPos})...)
}

// encodeSidecar writes era's envelope sidecar holding the envelopes of blocks with an applied payload.
func encodeSidecar(t *testing.T, era uint64, blocks []testBlock) []byte {
	t.Helper()

	start := (era - 1) * testSlotsPerEra
	file := e2Entry(typeVersion, nil)
	positions := make(map[uint64]int64, len(blocks))

	for _, b := range blocks {
		if b.envelope == nil {
			continue
		}

		positions[b.slot] = int64(len(file))
		file = append(file, e2Entry(typeCompressedEnvelope, snappyFramed(t, b.envelope))...)
	}

	idxPos := int64(len(file))
	offsets := make([]int64, testSlotsPerEra)

	for i := range offsets {
		if pos, ok := positions[start+uint64(i)]; ok {
			offsets[i] = pos - idxPos
		}
	}

	return append(file, e2SlotIndex(start, offsets)...)
}
