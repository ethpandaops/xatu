package blockarchive

import (
	"errors"
	"fmt"
	"sort"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// envelopeTailSize is the size of a sidecar's trailing slot index.
func envelopeTailSize(slotsPerEra uint64) int64 {
	return slotIndexSize(slotsPerEra)
}

// parseEnvelopeTail parses a sidecar's slot index, empty for slots without an applied payload.
func parseEnvelopeTail(tail []byte, size int64, era, slotsPerEra uint64) (*fileIndex, error) {
	if int64(len(tail)) != slotIndexSize(slotsPerEra) || size < int64(len(tail)) {
		return nil, fmt.Errorf("sidecar tail is %d bytes of a %d byte file", len(tail), size)
	}

	idx, err := parseSlotIndex(tail, size-int64(len(tail)))
	if err != nil {
		return nil, err
	}

	if idx.start != (era-1)*slotsPerEra || uint64(len(idx.offsets)) != slotsPerEra {
		return nil, fmt.Errorf("sidecar index covers %d slots from %d, want %d from %d",
			len(idx.offsets), idx.start, slotsPerEra, (era-1)*slotsPerEra)
	}

	if err := idx.validate(entryHeaderSize, idx.pos); err != nil {
		return nil, err
	}

	return &fileIndex{slots: idx, end: idx.pos}, nil
}

// decodeEnvelopes verifies envelopes against their blocks, returning every Gloas+ block's envelope (nil if withheld).
func decodeEnvelopes(
	chain *Chain,
	blocks map[uint64]*spec.VersionedSignedBeaconBlock,
	raw map[uint64][]byte,
) (map[uint64]*gloas.SignedExecutionPayloadEnvelope, error) {
	out := make(map[uint64]*gloas.SignedExecutionPayloadEnvelope, len(blocks))

	for slot, block := range blocks {
		if block != nil && hasEnvelopes(block.Version) {
			out[slot] = nil
		}
	}

	for slot := range raw {
		if _, ok := out[slot]; !ok {
			return nil, fmt.Errorf("slot %d: envelope without a gloas block", slot)
		}
	}

	decoded, err := decodeParallel(raw, func(slot uint64, data []byte) (*gloas.SignedExecutionPayloadEnvelope, error) {
		return decodeEnvelope(chain, blocks[slot], data)
	})
	if err != nil {
		return nil, err
	}

	for slot, envelope := range decoded {
		out[slot] = envelope
	}

	if err := checkPayloads(blocks, out); err != nil {
		return nil, err
	}

	return out, nil
}

func hasEnvelopes(v spec.DataVersion) bool {
	return v >= spec.DataVersionGloas
}

// decodeEnvelope decodes a block's envelope and checks it commits to the block and its bid.
func decodeEnvelope(chain *Chain, block *spec.VersionedSignedBeaconBlock, compressed []byte) (*gloas.SignedExecutionPayloadEnvelope, error) {
	ssz, err := decompress(compressed)
	if err != nil {
		return nil, err
	}

	envelope := &gloas.SignedExecutionPayloadEnvelope{}
	if err = chain.Codec.UnmarshalSSZ(envelope, ssz); err != nil {
		return nil, fmt.Errorf("failed to decode envelope: %w", err)
	}

	if envelope.Message == nil || envelope.Message.Payload == nil {
		return nil, errors.New("envelope has no payload")
	}

	message, err := blockMessage(block)
	if err != nil {
		return nil, err
	}

	root, err := chain.Codec.HashTreeRoot(message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash block: %w", err)
	}

	if envelope.Message.BeaconBlockRoot != root {
		return nil, fmt.Errorf("envelope commits to block %#x, not %#x", envelope.Message.BeaconBlockRoot, root)
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

func blockMessage(block *spec.VersionedSignedBeaconBlock) (any, error) {
	switch {
	case block.Gloas != nil && block.Gloas.Message != nil:
		return block.Gloas.Message, nil
	case block.Heze != nil && block.Heze.Message != nil:
		return block.Heze.Message, nil
	default:
		return nil, fmt.Errorf("%s block has no gloas message", block.Version)
	}
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

// checkPayloads checks each next bid builds on an applied payload's hash, or a withheld payload's parent hash.
func checkPayloads(blocks map[uint64]*spec.VersionedSignedBeaconBlock, envelopes map[uint64]*gloas.SignedExecutionPayloadEnvelope) error {
	slots := make([]uint64, 0, len(envelopes))
	for slot := range envelopes {
		slots = append(slots, slot)
	}

	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })

	var (
		prev                 uint64
		prevParent, prevHash phase0.Hash32
	)

	for i, slot := range slots {
		parent, hash, err := bidHashes(blocks[slot])
		if err != nil {
			return fmt.Errorf("slot %d: %w", slot, err)
		}

		if i > 0 {
			want := prevParent
			if envelopes[prev] != nil {
				want = prevHash
			}

			if parent != want {
				return fmt.Errorf("slot %d bid builds on %#x, which contradicts slot %d payload being %s",
					slot, parent, prev, payloadStatus(envelopes[prev]))
			}
		}

		prev, prevParent, prevHash = slot, parent, hash
	}

	return nil
}

func payloadStatus(envelope *gloas.SignedExecutionPayloadEnvelope) string {
	if envelope == nil {
		return "withheld"
	}

	return "applied"
}
