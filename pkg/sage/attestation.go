package sage

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/sage/event"
	"github.com/ethpandaops/xatu/pkg/sage/event/armiarma"
)

func (a *sage) handleAttestationEvent(ctx context.Context, e *armiarma.TimedEthereumAttestation) error {
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
