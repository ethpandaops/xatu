package sentry

import (
	"context"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	v1 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v1"
)

func (s *Sentry) startBeaconCommitteesWatcher(ctx context.Context) error {
	if !s.Config.BeaconCommittees.Enabled {
		return nil
	}

	s.beacon.OnReady(ctx, func(ctx context.Context) error {
		// Subscribe to future beacon committee events.
		s.beacon.Duties().OnBeaconCommittee(func(epoch phase0.Epoch, committees []*eth2v1.BeaconCommittee) error {
			// Ignore if the epoch is not the current one or the next one.
			currentEpoch := s.beacon.Metadata().Wallclock().Epochs().Current()
			if uint64(epoch) != currentEpoch.Number() && uint64(epoch) != currentEpoch.Number()+1 {
				return nil
			}

			if err := s.createNewBeaconCommitteeEvent(ctx, epoch, committees); err != nil {
				s.log.WithError(err).Error("Failed to create new beacon committee event")
			}

			return nil
		})

		// Grab the current epoch (+1) committees.
		now := s.beacon.Metadata().Wallclock().Epochs().Current()
		epochs := []phase0.Epoch{
			phase0.Epoch(now.Number()),
		}

		for _, epoch := range epochs {
			committees, err := s.beacon.Duties().GetAttestationDuties(epoch)
			if err != nil {
				s.log.WithError(err).Error("Failed to obtain beacon committees")

				continue
			}

			if err := s.createNewBeaconCommitteeEvent(ctx, epoch, committees); err != nil {
				s.log.WithError(err).Error("Failed to create new beacon committee event")
			}
		}

		return nil
	})

	return nil
}

func (s *Sentry) createNewBeaconCommitteeEvent(ctx context.Context, epoch phase0.Epoch, committees []*eth2v1.BeaconCommittee) error {
	now := time.Now()

	// Create an event for every committee.
	for _, committee := range committees {
		meta, err := s.createNewClientMeta(ctx)
		if err != nil {
			s.log.WithError(err).Error("Failed to create client meta when handling beacon committee event")

			continue
		}

		event := v1.NewBeaconCommittee(s.log, committee, epoch, now, s.beacon, meta, s.duplicateCache.BeaconEthV1BeaconCommittee)

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
