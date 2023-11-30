package seer

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/seer/event"
)

func (a *Seer) handleAttestationEvent(ctx context.Context, e *event.WrappedAttestation) error {
	meta, err := a.createNewClientMeta(ctx)
	if err != nil {
		return err
	}

	// Create a new hydrator
	hydrator := event.NewAttestationHydrator(a.log, e, a.beacon, a.duplicateCache.Attestation, meta)

	timesSeenBefore, err := hydrator.TimesSeen(ctx)
	if err != nil {
		return err
	}

	// If we've seen this event before and its over the threshold, ignore it
	if timesSeenBefore >= a.config.DuplicateAttestationThreshold {
		return nil
	}

	// Derive an event from the hydrator
	decoratedEvent, err := hydrator.CreateXatuEvent(ctx)
	if err != nil {
		return err
	}

	// Mark the event as seen
	if err := hydrator.MarkAttestationAsSeen(ctx); err != nil {
		a.log.WithError(err).Error("Failed to mark attestation as seen")

		return err
	}

	return a.handleNewDecoratedEvent(ctx, decoratedEvent)
}
