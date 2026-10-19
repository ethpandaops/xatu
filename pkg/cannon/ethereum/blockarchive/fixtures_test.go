package blockarchive

import (
	"bytes"
	"context"
	"encoding/binary"
	"sync/atomic"
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
	root     phase0.Root
	ssz      []byte
	envelope []byte
}

// testParent is the parent root of the first block of a test chain.
var testParent = phase0.Root{0x01}

// testBlocksAt builds a canonical chain with blocks at slots and every Gloas payload applied.
func testBlocksAt(t *testing.T, slots ...uint64) []testBlock {
	t.Helper()

	return linkedBlocks(t, testParent, slots, nil)
}

// gloasChain builds a canonical chain with blocks at slots, withholding the payloads of the withheld slots.
func gloasChain(t *testing.T, slots []uint64, withheld map[uint64]bool) []testBlock {
	t.Helper()

	return linkedBlocks(t, testParent, slots, withheld)
}

// linkedBlocks builds a chain of the forks' blocks at slots, linked by parent root and, from Gloas, by bid.
func linkedBlocks(t *testing.T, parent phase0.Root, slots []uint64, withheld map[uint64]bool) []testBlock {
	t.Helper()

	head := phase0.Hash32{0xaa}
	out := make([]testBlock, 0, len(slots))

	for _, slot := range slots {
		var b testBlock

		switch testChain().forkAt(slot) {
		case spec.DataVersionPhase0:
			b = phase0Block(t, slot, parent)
		case spec.DataVersionAltair:
			b = altairBlock(t, slot, parent)
		default:
			b = gloasBlock(t, slot, parent, head, !withheld[slot])

			if !withheld[slot] {
				head = testHash(slot)
			}
		}

		out = append(out, b)
		parent = b.root
	}

	return out
}

func phase0Block(t *testing.T, slot uint64, parent phase0.Root) testBlock {
	t.Helper()

	msg := &phase0.BeaconBlock{
		Slot:          phase0.Slot(slot),
		ProposerIndex: phase0.ValidatorIndex(slot % 97),
		ParentRoot:    parent,
		Body: &phase0.BeaconBlockBody{
			ETH1Data: &phase0.ETH1Data{BlockHash: make([]byte, 32)},
			Graffiti: [32]byte{'p', '0', byte(slot)},
		},
	}

	return encodeTestBlock(t, slot, msg, &phase0.SignedBeaconBlock{Message: msg})
}

func altairBlock(t *testing.T, slot uint64, parent phase0.Root) testBlock {
	t.Helper()

	msg := &altair.BeaconBlock{
		Slot:          phase0.Slot(slot),
		ProposerIndex: phase0.ValidatorIndex(slot % 89),
		ParentRoot:    parent,
		Body: &altair.BeaconBlockBody{
			ETH1Data:      &phase0.ETH1Data{BlockHash: make([]byte, 32)},
			Graffiti:      [32]byte{'a', 'l', byte(slot)},
			SyncAggregate: &altair.SyncAggregate{SyncCommitteeBits: make([]byte, 64)},
		},
	}

	return encodeTestBlock(t, slot, msg, &altair.SignedBeaconBlock{Message: msg})
}

// gloasBlock builds a Gloas block bidding testHash(slot) on parentHash, with its envelope when applied.
func gloasBlock(t *testing.T, slot uint64, parent phase0.Root, parentHash phase0.Hash32, applied bool) testBlock {
	t.Helper()

	msg := &gloas.BeaconBlock{
		Slot:          phase0.Slot(slot),
		ProposerIndex: phase0.ValidatorIndex(slot % 83),
		ParentRoot:    parent,
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

	b := encodeTestBlock(t, slot, msg, &gloas.SignedBeaconBlock{Message: msg})

	if applied {
		b.envelope = testEnvelope(t, b.root, testHash(slot))
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

// fakeBeacon answers block roots by slot for the blocks it is given, like the beacon node's headers endpoint.
type fakeBeacon struct {
	roots    map[uint64]phase0.Root
	err      error
	requests atomic.Int64
}

func newFakeBeacon(chains ...[]testBlock) *fakeBeacon {
	f := &fakeBeacon{roots: make(map[uint64]phase0.Root)}

	for _, blocks := range chains {
		for _, b := range blocks {
			f.roots[b.slot] = b.root
		}
	}

	return f
}

func (f *fakeBeacon) root(_ context.Context, slot phase0.Slot) (phase0.Root, bool, error) {
	f.requests.Add(1)

	if f.err != nil {
		return phase0.Root{}, false, f.err
	}

	root, ok := f.roots[uint64(slot)]

	return root, ok, nil
}

func encodeTestBlock(t *testing.T, slot uint64, message, signed any) testBlock {
	t.Helper()

	ssz, err := dynssz.GetGlobalDynSsz().MarshalSSZ(signed)
	require.NoError(t, err)

	root, err := dynssz.GetGlobalDynSsz().HashTreeRoot(message)
	require.NoError(t, err)

	return testBlock{slot: slot, root: root, ssz: ssz}
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
