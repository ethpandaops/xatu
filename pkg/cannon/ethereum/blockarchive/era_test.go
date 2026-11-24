package blockarchive

import (
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func eraIndexOf(t *testing.T, file []byte, era uint64) (*fileIndex, error) {
	t.Helper()

	size := eraTailSize(testSlotsPerEra)

	return parseEraTail(file[int64(len(file))-size:], int64(len(file)), era, testSlotsPerEra)
}

func TestChainForkAt(t *testing.T) {
	chain := testChain()

	tests := map[uint64]spec.DataVersion{
		0:                                     spec.DataVersionPhase0,
		testAltairEpoch*testSlotsPerEpoch - 1: spec.DataVersionPhase0,
		testAltairEpoch * testSlotsPerEpoch:   spec.DataVersionAltair,
		testGloasEpoch*testSlotsPerEpoch - 1:  spec.DataVersionAltair,
		testGloasEpoch * testSlotsPerEpoch:    spec.DataVersionGloas,
		1 << 30:                               spec.DataVersionGloas,
	}

	for slot, want := range tests {
		assert.Equal(t, want, chain.forkAt(slot), "slot %d", slot)
	}

	assert.Equal(t, uint64(1), chain.eraOf(0))
	assert.Equal(t, uint64(1), chain.eraOf(testSlotsPerEra-1))
	assert.Equal(t, uint64(2), chain.eraOf(testSlotsPerEra))
	assert.Equal(t, uint64(testSlotsPerEra), chain.eraStart(2))
}

func TestParseEraTail(t *testing.T) {
	for _, opts := range []eraOptions{{}, {nimbusMissed: true}, {unknownEntries: true}} {
		file := encodeEra(t, 2, testBlocksAt(t, 16, 17, 20, 31), opts)

		idx, err := eraIndexOf(t, file, 2)
		require.NoError(t, err)
		assert.Equal(t, uint64(16), idx.slots.start)
		assert.Len(t, idx.slots.offsets, testSlotsPerEra)

		pos, ok := idx.slots.entryAt(16)
		assert.True(t, ok)
		assert.Equal(t, int64(entryHeaderSize), pos, "the first block follows the version entry")

		for _, missed := range []uint64{18, 19, 21, 30} {
			_, ok = idx.slots.entryAt(missed)
			assert.False(t, ok, "slot %d is missed (%+v)", missed, opts)
		}

		assert.Greater(t, idx.end, pos, "entries end at the state")
	}
}

func TestParseEraTailRejectsMismatches(t *testing.T) {
	file := encodeEra(t, 2, testBlocksAt(t, 16, 17), eraOptions{})

	_, err := eraIndexOf(t, file, 3)
	require.ErrorContains(t, err, "want era 3", "the file must hold the era it is listed as")

	_, err = parseEraTail(file[len(file)-10:], int64(len(file)), 2, testSlotsPerEra)
	require.Error(t, err, "a short tail is rejected")

	// An era of a different length than the network's SLOTS_PER_HISTORICAL_ROOT.
	_, err = parseEraTail(file[len(file)-int(eraTailSize(8)):], int64(len(file)), 2, 8)
	require.Error(t, err)
}

func TestDecodeBlocksAcrossForkBoundary(t *testing.T) {
	chain := testChain()
	// Era 2 is slots 16..31 with altair activating at slot 24.
	blocks := testBlocksAt(t, 16, 23, 24, 31)
	file := encodeEra(t, 2, blocks, eraOptions{unknownEntries: true})

	idx, err := eraIndexOf(t, file, 2)
	require.NoError(t, err)

	lo, hi, ok := idx.slots.span(16, 31, idx.end)
	require.True(t, ok)

	raw, err := indexedEntries(idx.slots, typeCompressedSignedBeaconBlock, 16, 31, file[lo:hi], lo)
	require.NoError(t, err)
	require.Len(t, raw, 4)

	decoded, err := decodeBlocks(chain, raw)
	require.NoError(t, err)
	require.NoError(t, checkLinks(decoded))

	for _, b := range blocks {
		block := decoded[b.slot].block
		require.NotNil(t, block, "slot %d", b.slot)
		assert.Equal(t, chain.forkAt(b.slot), block.Version, "slot %d", b.slot)
		assert.Equal(t, b.root, decoded[b.slot].root, "slot %d", b.slot)

		got, err := chain.Codec.MarshalSSZ(versionedMessage(t, block))
		require.NoError(t, err)
		assert.Equal(t, b.ssz, got, "slot %d round trips byte for byte", b.slot)
	}
}

func TestCheckLinks(t *testing.T) {
	chain := testChain()
	blocks := testBlocksAt(t, 16, 17, 20, 24)
	other := linkedBlocks(t, phase0.Root{0x02}, []uint64{17}, nil)

	decode := func(bs ...testBlock) map[uint64]*decodedBlock {
		raw := make(map[uint64][]byte, len(bs))
		for _, b := range bs {
			raw[b.slot] = snappyFramed(t, b.ssz)
		}

		decoded, err := decodeBlocks(chain, raw)
		require.NoError(t, err)

		return decoded
	}

	require.NoError(t, checkLinks(decode(blocks...)))
	require.NoError(t, checkLinks(decode(blocks[2])), "a lone block links trivially")
	require.ErrorContains(t, checkLinks(decode(blocks[0], blocks[2])), "slot 20 builds on", "a block left out breaks the links")
	require.ErrorContains(t, checkLinks(decode(blocks[0], other[0], blocks[2])), "slot 17 builds on")
}

func TestDecodeBlockChecksIndexSlot(t *testing.T) {
	chain := testChain()
	b := phase0Block(t, 17, testParent)

	_, err := decodeBlock(chain, 18, snappyFramed(t, b.ssz))
	require.ErrorContains(t, err, "decodes to slot 17")

	// A block indexed in the wrong fork's slots does not decode as that fork.
	_, err = decodeBlock(chain, 24, snappyFramed(t, b.ssz))
	require.Error(t, err)
}

func versionedMessage(t *testing.T, block *spec.VersionedSignedBeaconBlock) any {
	t.Helper()

	switch block.Version {
	case spec.DataVersionPhase0:
		return block.Phase0
	case spec.DataVersionAltair:
		return block.Altair
	case spec.DataVersionGloas:
		return block.Gloas
	default:
		t.Fatalf("unexpected version %s", block.Version)

		return nil
	}
}
