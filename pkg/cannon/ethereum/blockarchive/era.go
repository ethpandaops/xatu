package blockarchive

import (
	"errors"
	"fmt"
	"runtime"

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

// decodeParallel decodes each slot's raw entry with fn across GOMAXPROCS workers.
func decodeParallel[T any](raw map[uint64][]byte, fn func(slot uint64, data []byte) (T, error)) (map[uint64]T, error) {
	slots := make([]uint64, 0, len(raw))
	for slot := range raw {
		slots = append(slots, slot)
	}

	decoded := make([]T, len(slots))

	g := errgroup.Group{}
	g.SetLimit(runtime.GOMAXPROCS(0))

	for i, slot := range slots {
		g.Go(func() error {
			v, err := fn(slot, raw[slot])
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

	out := make(map[uint64]T, len(slots))
	for i, slot := range slots {
		out[slot] = decoded[i]
	}

	return out, nil
}

// decodeBlocks decodes compressed block entries, checking each block's slot matches its index slot.
func decodeBlocks(chain *Chain, raw map[uint64][]byte) (map[uint64]*spec.VersionedSignedBeaconBlock, error) {
	return decodeParallel(raw, func(slot uint64, data []byte) (*spec.VersionedSignedBeaconBlock, error) {
		return decodeBlock(chain, slot, data)
	})
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
