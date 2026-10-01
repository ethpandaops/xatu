package v2

import (
	"context"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
)

// DefaultBlockAccessListMaxRowsPerBatch is the default upper bound on the
// number of events handed to the output sinks in a single call.
const DefaultBlockAccessListMaxRowsPerBatch = 50_000

type (
	slotDeriveFunc func(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error)
	eventSendFunc  func(ctx context.Context, events []*xatu.DecoratedEvent) error
)

// epochStreamer delivers the events of one epoch to a sender in bounded
// batches instead of as a single epoch-sized slice.
//
// Slots are derived one at a time and whole slots are packed into a batch
// until the next slot would push it past maxRows. A slot that alone exceeds
// maxRows is split into consecutive batches of at most maxRows. No call to
// the sender ever carries more than maxRows events, and low volume epochs
// still collapse into a single batch.
//
// stream returns nil only once the sender has accepted every batch of the
// epoch. If it fails part way, the last slot boundary at which everything
// before it was accepted is remembered, and a repeat call for the same epoch
// resumes from there. Slots before that boundary are neither re-derived nor
// re-sent. The slots after it, which include the batch that failed, are sent
// again in full, so the sender must tolerate replays of a partially applied
// batch.
//
// An epochStreamer is not safe for concurrent use.
type epochStreamer struct {
	maxRows int
	derive  slotDeriveFunc
	send    eventSendFunc

	resuming    bool
	resumeEpoch phase0.Epoch
	// resumeSlot is the first slot of resumeEpoch whose events the sender has
	// not fully accepted.
	resumeSlot phase0.Slot
}

// newEpochStreamer returns a streamer that sends at most maxRows events per
// call. A non-positive maxRows falls back to the default.
func newEpochStreamer(maxRows int, derive slotDeriveFunc, send eventSendFunc) *epochStreamer {
	if maxRows < 1 {
		maxRows = DefaultBlockAccessListMaxRowsPerBatch
	}

	return &epochStreamer{
		maxRows: maxRows,
		derive:  derive,
		send:    send,
	}
}

// reset forgets any partial progress. Callers invoke it once the epoch's
// checkpoint has been persisted, so that deriving the same epoch again later
// (for example after an operator rewinds the checkpoint) emits it afresh.
func (s *epochStreamer) reset() {
	s.resuming = false
}

// stream derives slots [firstSlot, firstSlot+numSlots) and sends their events
// in bounded batches, in slot order.
func (s *epochStreamer) stream(
	ctx context.Context,
	epoch phase0.Epoch,
	firstSlot phase0.Slot,
	numSlots uint64,
) error {
	endSlot := firstSlot + phase0.Slot(numSlots)

	startSlot := firstSlot
	if s.resuming && s.resumeEpoch == epoch && s.resumeSlot > startSlot {
		startSlot = min(s.resumeSlot, endSlot)
	}

	var pending []*xatu.DecoratedEvent

	// flush sends everything pending, which covers the slots before throughSlot.
	flush := func(throughSlot phase0.Slot) error {
		if err := s.sendBatches(ctx, pending); err != nil {
			return err
		}

		pending = nil
		s.resuming, s.resumeEpoch, s.resumeSlot = true, epoch, throughSlot

		return nil
	}

	for slot := startSlot; slot < endSlot; slot++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		events, err := s.derive(ctx, slot)
		if err != nil {
			return errors.Wrapf(err, "failed to process slot %d", slot)
		}

		if len(pending) > 0 && len(pending)+len(events) > s.maxRows {
			if err := flush(slot); err != nil {
				return err
			}
		}

		pending = append(pending, events...)

		if len(pending) >= s.maxRows {
			if err := flush(slot + 1); err != nil {
				return err
			}
		}
	}

	return flush(endSlot)
}

func (s *epochStreamer) sendBatches(ctx context.Context, events []*xatu.DecoratedEvent) error {
	for len(events) > 0 {
		n := min(len(events), s.maxRows)

		if err := s.send(ctx, events[:n:n]); err != nil {
			return err
		}

		events = events[n:]
	}

	return nil
}
