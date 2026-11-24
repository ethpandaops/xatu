package blockarchive

import (
	"errors"
	"fmt"
	"runtime"
	"slices"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/heze"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	dynssz "github.com/pk910/dynamic-ssz"
	"golang.org/x/sync/errgroup"
)

// Chain is the network configuration archived blocks are decoded with.
type Chain struct {
	Codec         *dynssz.DynSsz
	SlotsPerEpoch uint64
	// SlotsPerEra is SLOTS_PER_HISTORICAL_ROOT, the number of slots in one era.
	SlotsPerEra uint64
	Forks       state.ForkEpochs
}

func (c *Chain) validate() error {
	if c.Codec == nil || c.SlotsPerEpoch == 0 || c.SlotsPerEra == 0 {
		return errors.New("incomplete chain configuration")
	}

	return nil
}

// forkAt returns the fork active at slot.
func (c *Chain) forkAt(slot uint64) spec.DataVersion {
	epoch := phase0.Epoch(slot / c.SlotsPerEpoch)
	version := spec.DataVersionPhase0

	for _, f := range c.Forks {
		if f.Epoch <= epoch && f.Name > version {
			version = f.Name
		}
	}

	return version
}

// eraOf returns the era whose file holds slot's block: era N holds slots [(N-1)*SlotsPerEra, N*SlotsPerEra).
func (c *Chain) eraOf(slot uint64) uint64 {
	return slot/c.SlotsPerEra + 1
}

func (c *Chain) eraStart(era uint64) uint64 {
	return (era - 1) * c.SlotsPerEra
}

// fileIndex locates the slot-indexed entries of an era file or envelope sidecar.
type fileIndex struct {
	slots *slotIndex
	// end is where the indexed entries end: the era's state entry, or the sidecar's slot index.
	end int64
}

// eraTailSize is the size of an era file's trailing block and state slot indexes.
func eraTailSize(slotsPerEra uint64) int64 {
	return slotIndexSize(slotsPerEra) + slotIndexSize(1)
}

// parseEraTail parses the block and state slot indexes ending a single-group era file.
func parseEraTail(tail []byte, size int64, era, slotsPerEra uint64) (*fileIndex, error) {
	stateIdxSize, blockIdxSize := slotIndexSize(1), slotIndexSize(slotsPerEra)
	if int64(len(tail)) != stateIdxSize+blockIdxSize || size < int64(len(tail)) {
		return nil, fmt.Errorf("era tail is %d bytes of a %d byte file", len(tail), size)
	}

	stateIdx, err := parseSlotIndex(tail[blockIdxSize:], size-stateIdxSize)
	if err != nil {
		return nil, fmt.Errorf("state index: %w", err)
	}

	if stateIdx.start != era*slotsPerEra {
		return nil, fmt.Errorf("state index is for slot %d, want era %d slot %d", stateIdx.start, era, era*slotsPerEra)
	}

	statePos, ok := stateIdx.entryAt(stateIdx.start)
	if !ok {
		return nil, errors.New("state index has no state entry")
	}

	blockIdx, err := parseSlotIndex(tail[:blockIdxSize], size-stateIdxSize-blockIdxSize)
	if err != nil {
		return nil, fmt.Errorf("block index: %w", err)
	}

	if blockIdx.start != (era-1)*slotsPerEra || uint64(len(blockIdx.offsets)) != slotsPerEra {
		return nil, fmt.Errorf("block index covers %d slots from %d, want %d from %d",
			len(blockIdx.offsets), blockIdx.start, slotsPerEra, (era-1)*slotsPerEra)
	}

	if statePos >= blockIdx.pos {
		return nil, fmt.Errorf("state entry offset %d is past the block index", statePos)
	}

	if err := blockIdx.validate(entryHeaderSize, statePos); err != nil {
		return nil, fmt.Errorf("block index: %w", err)
	}

	return &fileIndex{slots: blockIdx, end: statePos}, nil
}

// decodeParallel applies fn to each slot's input across GOMAXPROCS workers.
func decodeParallel[In, Out any](in map[uint64]In, fn func(slot uint64, v In) (Out, error)) (map[uint64]Out, error) {
	slots := make([]uint64, 0, len(in))
	for slot := range in {
		slots = append(slots, slot)
	}

	decoded := make([]Out, len(slots))

	g := errgroup.Group{}
	g.SetLimit(runtime.GOMAXPROCS(0))

	for i, slot := range slots {
		g.Go(func() error {
			v, err := fn(slot, in[slot])
			if err != nil {
				return fmt.Errorf("slot %d: %w", slot, err)
			}

			decoded[i] = v

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	out := make(map[uint64]Out, len(slots))
	for i, slot := range slots {
		out[slot] = decoded[i]
	}

	return out, nil
}

// decodedBlock is a block decoded from an era file with its root and signature.
type decodedBlock struct {
	block *spec.VersionedSignedBeaconBlock
	root  phase0.Root
	sig   phase0.BLSSignature
}

// decodeBlocks decodes and hashes compressed block entries, checking each block's slot matches its index slot.
func decodeBlocks(chain *Chain, raw map[uint64][]byte) (map[uint64]*decodedBlock, error) {
	return decodeParallel(raw, func(slot uint64, data []byte) (*decodedBlock, error) {
		block, err := decodeBlock(chain, slot, data)
		if err != nil {
			return nil, err
		}

		message, sig, err := blockParts(block)
		if err != nil {
			return nil, err
		}

		root, err := chain.Codec.HashTreeRoot(message)
		if err != nil {
			return nil, fmt.Errorf("failed to hash block: %w", err)
		}

		return &decodedBlock{block: block, root: root, sig: sig}, nil
	})
}

// checkLinks checks each decoded block's parent root is the root of the decoded block before it.
func checkLinks(blocks map[uint64]*decodedBlock) error {
	slots := make([]uint64, 0, len(blocks))
	for slot := range blocks {
		slots = append(slots, slot)
	}

	slices.Sort(slots)

	for i := 1; i < len(slots); i++ {
		parent, err := blocks[slots[i]].block.ParentRoot()
		if err != nil {
			return fmt.Errorf("slot %d: %w", slots[i], err)
		}

		if parent != blocks[slots[i-1]].root {
			return fmt.Errorf("slot %d builds on %#x, not on slot %d block %#x", slots[i], parent, slots[i-1], blocks[slots[i-1]].root)
		}
	}

	return nil
}

func decodeBlock(chain *Chain, slot uint64, compressed []byte) (*spec.VersionedSignedBeaconBlock, error) {
	ssz, err := decompress(compressed)
	if err != nil {
		return nil, err
	}

	block, err := unmarshalBlock(chain.Codec, chain.forkAt(slot), ssz)
	if err != nil {
		return nil, err
	}

	got, err := block.Slot()
	if err != nil {
		return nil, err
	}

	if uint64(got) != slot {
		return nil, fmt.Errorf("block decodes to slot %d", got)
	}

	return block, nil
}

// blockParts returns the BeaconBlock a signed block wraps and its signature.
func blockParts(block *spec.VersionedSignedBeaconBlock) (message any, signature phase0.BLSSignature, err error) {
	switch block.Version {
	case spec.DataVersionPhase0:
		if block.Phase0 != nil && block.Phase0.Message != nil {
			return block.Phase0.Message, block.Phase0.Signature, nil
		}
	case spec.DataVersionAltair:
		if block.Altair != nil && block.Altair.Message != nil {
			return block.Altair.Message, block.Altair.Signature, nil
		}
	case spec.DataVersionBellatrix:
		if block.Bellatrix != nil && block.Bellatrix.Message != nil {
			return block.Bellatrix.Message, block.Bellatrix.Signature, nil
		}
	case spec.DataVersionCapella:
		if block.Capella != nil && block.Capella.Message != nil {
			return block.Capella.Message, block.Capella.Signature, nil
		}
	case spec.DataVersionDeneb:
		if block.Deneb != nil && block.Deneb.Message != nil {
			return block.Deneb.Message, block.Deneb.Signature, nil
		}
	case spec.DataVersionElectra:
		if block.Electra != nil && block.Electra.Message != nil {
			return block.Electra.Message, block.Electra.Signature, nil
		}
	case spec.DataVersionFulu:
		if block.Fulu != nil && block.Fulu.Message != nil {
			return block.Fulu.Message, block.Fulu.Signature, nil
		}
	case spec.DataVersionGloas:
		if block.Gloas != nil && block.Gloas.Message != nil {
			return block.Gloas.Message, block.Gloas.Signature, nil
		}
	case spec.DataVersionHeze:
		if block.Heze != nil && block.Heze.Message != nil {
			return block.Heze.Message, block.Heze.Signature, nil
		}
	default:
	}

	return nil, signature, fmt.Errorf("%s block has no message", block.Version)
}

// unmarshalBlock decodes a SignedBeaconBlock of fork v as go-eth2-client decodes SSZ API responses.
func unmarshalBlock(ds *dynssz.DynSsz, v spec.DataVersion, ssz []byte) (*spec.VersionedSignedBeaconBlock, error) {
	block := &spec.VersionedSignedBeaconBlock{Version: v}

	var target any

	switch v {
	case spec.DataVersionPhase0:
		block.Phase0 = &phase0.SignedBeaconBlock{}
		target = block.Phase0
	case spec.DataVersionAltair:
		block.Altair = &altair.SignedBeaconBlock{}
		target = block.Altair
	case spec.DataVersionBellatrix:
		block.Bellatrix = &bellatrix.SignedBeaconBlock{}
		target = block.Bellatrix
	case spec.DataVersionCapella:
		block.Capella = &capella.SignedBeaconBlock{}
		target = block.Capella
	case spec.DataVersionDeneb:
		block.Deneb = &deneb.SignedBeaconBlock{}
		target = block.Deneb
	case spec.DataVersionElectra:
		block.Electra = &electra.SignedBeaconBlock{}
		target = block.Electra
	case spec.DataVersionFulu:
		block.Fulu = &electra.SignedBeaconBlock{}
		target = block.Fulu
	case spec.DataVersionGloas:
		block.Gloas = &gloas.SignedBeaconBlock{}
		target = block.Gloas
	case spec.DataVersionHeze:
		block.Heze = &heze.SignedBeaconBlock{}
		target = block.Heze
	default:
		return nil, fmt.Errorf("unhandled block fork %s", v)
	}

	if err := ds.UnmarshalSSZ(target, ssz); err != nil {
		return nil, fmt.Errorf("failed to decode %s block: %w", v, err)
	}

	return block, nil
}
