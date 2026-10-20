package iterator

import (
	"context"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// EpochDeriveFunc derives the events of one epoch.
type EpochDeriveFunc func(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error)

// DeriveEpochs derives every epoch in order and returns all their events, or none if any epoch fails.
func DeriveEpochs(ctx context.Context, epochs []phase0.Epoch, derive EpochDeriveFunc) ([]*xatu.DecoratedEvent, error) {
	if len(epochs) == 0 {
		return nil, errors.New("no epochs to derive")
	}

	if len(epochs) == 1 {
		return derive(ctx, epochs[0])
	}

	var all []*xatu.DecoratedEvent

	for _, epoch := range epochs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		events, err := derive(ctx, epoch)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to derive epoch %d", epoch)
		}

		all = append(all, events...)
	}

	return all, nil
}

// backfillEpochs walks down from next for up to size epochs, never below target (callers guarantee next >= target).
func backfillEpochs(next, target phase0.Epoch, size uint64) []phase0.Epoch {
	size = max(1, size)

	lo := target
	if uint64(next-target) >= size-1 {
		lo = next - phase0.Epoch(size-1)
	}

	epochs := make([]phase0.Epoch, 0, uint64(next-lo)+1)
	for e := next; ; e-- {
		epochs = append(epochs, e)

		if e == lo {
			break
		}
	}

	return epochs
}

// backfillLookAheads lists up to count epochs from next downward, never below target (callers guarantee next >= target).
func backfillLookAheads(next, target phase0.Epoch, count uint64) []phase0.Epoch {
	span := count
	if above := uint64(next - target); above < count {
		span = above + 1
	}

	epochs := make([]phase0.Epoch, 0, span)
	for i := range span {
		epochs = append(epochs, next-phase0.Epoch(i))
	}

	return epochs
}
