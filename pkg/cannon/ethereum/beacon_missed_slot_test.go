package ethereum

import (
	"context"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBeaconBlockConfirmsMissedSlots(t *testing.T) {
	parent := phase0.Root{0x64}

	// child is the block at slot 102, whose parent is the block with root parent.
	child := &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionPhase0,
		Phase0:  &phase0.SignedBeaconBlock{Message: &phase0.BeaconBlock{Slot: 102, ParentRoot: parent}},
	}

	tests := []struct {
		name        string
		blocks      map[string]*spec.VersionedSignedBeaconBlock
		headerSlots map[string]phase0.Slot
		wantErr     string
	}{
		{
			name:        "next block builds on an older block",
			blocks:      map[string]*spec.VersionedSignedBeaconBlock{"102": child},
			headerSlots: map[string]phase0.Slot{parent.String(): 100},
		},
		{
			// A node whose history starts after slot 101 404s it and does not have the next block's parent.
			name:    "history does not reach the slot",
			blocks:  map[string]*spec.VersionedSignedBeaconBlock{"102": child},
			wantErr: "block history does not reach slot 101",
		},
		{
			name:        "next block builds on the slot",
			blocks:      map[string]*spec.VersionedSignedBeaconBlock{"102": child},
			headerSlots: map[string]phase0.Slot{parent.String(): 101},
			wantErr:     "builds on a block at slot 101",
		},
		{
			name:        "no block after the slot",
			blocks:      map[string]*spec.VersionedSignedBeaconBlock{},
			headerSlots: map[string]phase0.Slot{"head": 101},
			wantErr:     "no block after slot 101 up to its head at slot 101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, node := newArchiveTestNode(nil)
			node.blocks = tt.blocks
			node.headerSlots = tt.headerSlots

			for range 2 {
				block, err := b.GetBeaconBlock(context.Background(), "101")
				assert.Nil(t, block)

				if tt.wantErr == "" {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, tt.wantErr)
				}
			}

			// A confirmed missed slot is cached; an unconfirmed 404 is asked again.
			fetches := 0

			for _, id := range node.fetched {
				if id == "101" {
					fetches++
				}
			}

			if tt.wantErr == "" {
				assert.Equal(t, 1, fetches)
			} else {
				assert.Equal(t, 2, fetches)
			}
		})
	}
}

func TestGetBeaconBlockLeavesRootNotFoundUnchecked(t *testing.T) {
	b, node := newArchiveTestNode(nil)
	node.blocks = map[string]*spec.VersionedSignedBeaconBlock{}

	block, err := b.GetBeaconBlock(context.Background(), phase0.Root{0x01}.String())
	require.NoError(t, err)
	assert.Nil(t, block)
	assert.Len(t, node.fetched, 1)
}

// A chain stall leaves more empty slots after a block than the lookahead scans; the slot is confirmed from a later
// block's ancestry.
func TestGetBeaconBlockConfirmsSlotsInALongGap(t *testing.T) {
	beforeGap, firstAfter, later := phase0.Root{0x01}, phase0.Root{0x02}, phase0.Root{0x03}

	block := func(slot phase0.Slot, parent phase0.Root) *spec.VersionedSignedBeaconBlock {
		return &spec.VersionedSignedBeaconBlock{
			Version: spec.DataVersionPhase0,
			Phase0:  &phase0.SignedBeaconBlock{Message: &phase0.BeaconBlock{Slot: slot, ParentRoot: parent}},
		}
	}

	b, node := newArchiveTestNode(nil)
	// Slots 101-499 are empty: 500 builds on 100, and 520 on 500.
	node.blocks = map[string]*spec.VersionedSignedBeaconBlock{
		"500": block(500, beforeGap),
		"520": block(520, firstAfter),
	}
	node.headerSlots = map[string]phase0.Slot{beforeGap.String(): 100, firstAfter.String(): 500, later.String(): 520, "head": 520}
	node.headerParents = map[string]phase0.Root{firstAfter.String(): beforeGap, later.String(): firstAfter}

	for _, slot := range []string{"101", "300", "499"} {
		missed, err := b.GetBeaconBlock(context.Background(), slot)
		require.NoError(t, err, "slot %s", slot)
		assert.Nil(t, missed)
	}

	next, slot, parentSlot, err := b.nextCanonicalBlock(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, uint64(500), slot)
	assert.Equal(t, phase0.Slot(100), parentSlot)
	assert.Same(t, node.blocks["500"], next)
}
