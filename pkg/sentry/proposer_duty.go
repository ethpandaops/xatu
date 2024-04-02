package sentry

import (
	"context"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	v1 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v1"
)

func (s *Sentry) startProposerDutyWatcher(ctx context.Context) error {
	if !s.Config.ProposerDuty.Enabled {
		return nil
	}

	// Subscribe to future proposer duty events.
	s.beacon.Duties().OnProposerDuties(func(epoch phase0.Epoch, duties []*eth2v1.ProposerDuty) error {
		if err := s.createNewProposerDutyEvent(ctx, epoch, duties); err != nil {
			s.log.WithError(err).Error("Failed to create new proposer duties event")
		}

		return nil
	})

	s.beacon.OnReady(ctx, func(ctx context.Context) error {
		// Grab the current epoch duties.
		now := s.beacon.Metadata().Wallclock().Epochs().Current()
		epochs := []phase0.Epoch{
			phase0.Epoch(now.Number()),
		}

		for _, epoch := range epochs {
			duties, err := s.beacon.Duties().GetProposerDuties(epoch)
			if err != nil {
				s.log.WithError(err).Error("Failed to obtain proposer duties")

				continue
			}

			if err := s.createNewProposerDutyEvent(ctx, epoch, duties); err != nil {
				s.log.WithError(err).Error("Failed to create new proposer duties event")
			}
		}

		return nil
	})

	return nil
}

func (s *Sentry) createNewProposerDutyEvent(ctx context.Context, epoch phase0.Epoch, duties []*eth2v1.ProposerDuty) error {
	now := time.Now()

	// Create an event for every duty.
	for _, duty := range duties {
		meta, err := s.createNewClientMeta(ctx)
		if err != nil {
			s.log.WithError(err).Error("Failed to create client meta when handling beacon committee event")

			continue
		}

		event := v1.NewProposerDuty(s.log, duty, epoch, now, s.beacon, meta, s.duplicateCache.BeaconEthV1BeaconCommittee)

		decoratedEvent, err := event.Decorate(ctx)
		if err != nil {
			s.log.WithError(err).Error("Failed to decorate beacon committee event")

			return err
		}

		if err := s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
			s.log.WithError(err).Error("Failed to handle decorated beacon committee event")
		}
	}

	return nil
}
