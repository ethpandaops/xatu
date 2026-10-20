package iterator

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func epochRange(hi, lo phase0.Epoch) []phase0.Epoch {
	out := make([]phase0.Epoch, 0, hi-lo+1)
	for e := hi; ; e-- {
		out = append(out, e)

		if e == lo {
			return out
		}
	}
}

func TestBackfillEpochs(t *testing.T) {
	tests := []struct {
		name   string
		next   phase0.Epoch
		target phase0.Epoch
		size   uint64
		want   []phase0.Epoch
	}{
		{name: "size one is a single epoch", next: 500, target: 0, size: 1, want: []phase0.Epoch{500}},
		{name: "size zero is a single epoch", next: 500, target: 0, size: 0, want: []phase0.Epoch{500}},
		{name: "full batch walks down", next: 500, target: 0, size: 4, want: []phase0.Epoch{500, 499, 498, 497}},
		{name: "batch ending exactly on target", next: 103, target: 100, size: 4, want: []phase0.Epoch{103, 102, 101, 100}},
		{name: "batch clamped at target", next: 102, target: 100, size: 64, want: []phase0.Epoch{102, 101, 100}},
		{name: "next on target is one epoch", next: 100, target: 100, size: 64, want: []phase0.Epoch{100}},
		{name: "batch clamped at genesis", next: 2, target: 0, size: 64, want: []phase0.Epoch{2, 1, 0}},
		{name: "genesis alone", next: 0, target: 0, size: 64, want: []phase0.Epoch{0}},
		{name: "batch clamped at a fork epoch", next: 269570, target: 269568, size: 8, want: []phase0.Epoch{269570, 269569, 269568}},
		{
			name: "huge next does not overflow", next: math.MaxUint64 - 1, target: 0, size: 3,
			want: []phase0.Epoch{math.MaxUint64 - 1, math.MaxUint64 - 2, math.MaxUint64 - 3},
		},
		{name: "max batch", next: 1000, target: 0, size: MaxEpochsPerBatch, want: epochRange(1000, 1000-MaxEpochsPerBatch+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, backfillEpochs(tt.next, tt.target, tt.size))
		})
	}
}

func TestBackfillLookAheads(t *testing.T) {
	tests := []struct {
		name   string
		next   phase0.Epoch
		target phase0.Epoch
		count  uint64
		want   []phase0.Epoch
	}{
		{name: "three epochs from next", next: 500, target: 0, count: 3, want: []phase0.Epoch{500, 499, 498}},
		{name: "clamped at target", next: 101, target: 100, count: 3, want: []phase0.Epoch{101, 100}},
		{name: "clamped at genesis", next: 1, target: 0, count: 128, want: []phase0.Epoch{1, 0}},
		{name: "zero count", next: 500, target: 0, count: 0, want: []phase0.Epoch{}},
		{name: "two batches of 64", next: 1000, target: 0, count: 128, want: epochRange(1000, 873)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, backfillLookAheads(tt.next, tt.target, tt.count))
		})
	}
}

func epochEvent(epoch phase0.Epoch) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{Event: &xatu.Event{Id: epochID(epoch)}}
}

func TestDeriveEpochs(t *testing.T) {
	ctx := context.Background()

	var derived []phase0.Epoch

	derive := func(_ context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
		derived = append(derived, epoch)

		return []*xatu.DecoratedEvent{epochEvent(epoch), epochEvent(epoch)}, nil
	}

	t.Run("derives in order and concatenates", func(t *testing.T) {
		derived = nil

		events, err := DeriveEpochs(ctx, []phase0.Epoch{10, 9, 8}, derive)
		require.NoError(t, err)
		assert.Equal(t, []phase0.Epoch{10, 9, 8}, derived)
		require.Len(t, events, 6)
		assert.Equal(t, epochID(10), events[0].GetEvent().GetId())
		assert.Equal(t, epochID(8), events[5].GetEvent().GetId())
	})

	t.Run("no epochs is an error", func(t *testing.T) {
		_, err := DeriveEpochs(ctx, nil, derive)
		require.Error(t, err)
	})

	errBoom := errors.New("boom")

	t.Run("failure mid batch discards every event", func(t *testing.T) {
		derived = nil

		failing := func(c context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
			if epoch == 9 {
				return nil, errBoom
			}

			return derive(c, epoch)
		}

		events, err := DeriveEpochs(ctx, []phase0.Epoch{10, 9, 8}, failing)
		require.ErrorIs(t, err, errBoom)
		assert.Contains(t, err.Error(), "epoch 9")
		assert.Nil(t, events)
		assert.Equal(t, []phase0.Epoch{10}, derived, "epochs after the failure are not derived")
	})

	t.Run("single epoch error is passed through", func(t *testing.T) {
		failing := func(context.Context, phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
			return nil, errBoom
		}

		_, err := DeriveEpochs(ctx, []phase0.Epoch{10}, failing)
		assert.Equal(t, errBoom, err)
	})

	t.Run("cancelled context stops the batch", func(t *testing.T) {
		derived = nil

		cctx, cancel := context.WithCancel(ctx)

		cancelling := func(c context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
			cancel()

			return derive(c, epoch)
		}

		events, err := DeriveEpochs(cctx, []phase0.Epoch{10, 9, 8}, cancelling)
		require.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, events)
		assert.Equal(t, []phase0.Epoch{10}, derived)
	})
}

func TestNextResponseEpochs(t *testing.T) {
	legacy := &BackFillingCheckpointNextResponse{Next: 42, Direction: BackfillingCheckpointDirectionHead}
	assert.Equal(t, []phase0.Epoch{42}, legacy.Epochs())
	assert.Equal(t, phase0.Epoch(42), legacy.LastEpoch())

	batch := &BackFillingCheckpointNextResponse{
		Next:      42,
		Direction: BackfillingCheckpointDirectionBackfill,
		epochs:    []phase0.Epoch{42, 41, 40},
	}
	assert.Equal(t, []phase0.Epoch{42, 41, 40}, batch.Epochs())
	assert.Equal(t, phase0.Epoch(40), batch.LastEpoch())
}

func TestBackfillingCheckpointConfig(t *testing.T) {
	cfg := &BackfillingCheckpointConfig{}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, uint64(1), cfg.epochsPerBatch(), "unset batch size is one epoch")

	cfg.Backfill.EpochsPerBatch = MaxEpochsPerBatch
	require.NoError(t, cfg.Validate())
	assert.Equal(t, uint64(MaxEpochsPerBatch), cfg.epochsPerBatch())

	cfg.Backfill.EpochsPerBatch = MaxEpochsPerBatch + 1
	require.Error(t, cfg.Validate())
}
