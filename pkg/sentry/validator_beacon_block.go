package sentry

import (
	"context"
	"fmt"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	v2 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v2"
)

func (s *Sentry) startValidatorsBeaconBlockSchedule(ctx context.Context) error {
	if !s.Config.ValidatorsBeaconBlock.Enabled {
		return nil
	}

	if s.Config.ValidatorsBeaconBlock.Interval.Enabled {
		logCtx := s.log.WithField("proccer", "interval").WithField("interval", s.Config.ValidatorsBeaconBlock.Interval.Every.String())

		if _, err := s.scheduler.Every(s.Config.ValidatorsBeaconBlock.Interval.Every.Duration).Do(func() {
			logCtx.Debug("Fetching validator beacon block")

			err := s.fetchDecoratedValidatorBeaconBlock(ctx)
			if err != nil {
				logCtx.WithError(err).Error("Failed to fetch validator beacon block")
			}
		}); err != nil {
			return err
		}
	}

	if s.Config.ValidatorsBeaconBlock.At.Enabled {
		for _, slotTime := range s.Config.ValidatorsBeaconBlock.At.SlotTimes {
			s.scheduleValidatorBeaconBlockFetchingAtSlotTime(ctx, slotTime.Duration)
		}
	}

	return nil
}

func (s *Sentry) scheduleValidatorBeaconBlockFetchingAtSlotTime(ctx context.Context, at time.Duration) {
	offset := at

	logCtx := s.log.
		WithField("proccer", "at_slot_time").
		WithField("slot_time", offset.String())

	logCtx.Debug("Scheduling validator beacon block fetching at slot time")

	s.beacon.Metadata().Wallclock().OnSlotChanged(func(slot ethwallclock.Slot) {
		time.Sleep(offset)

		logCtx.WithField("slot", slot.Number()).Debug("Fetching validator beacon block")

		if err := s.fetchDecoratedValidatorBeaconBlock(ctx); err != nil {
			logCtx.WithField("slot_time", offset.String()).WithError(err).Error("Failed to fetch validator beacon block")
		}
	})
}

func (s *Sentry) fetchValidatorBeaconBlock(ctx context.Context) (*v2.ValidatorBeaconBlock, error) {
	snapshot := &v2.ValidatorBeaconBlockDataSnapshot{
		RequestAt: time.Now(),
	}

	slot, _, err := s.beacon.Metadata().Wallclock().Now()
	if err != nil {
		return nil, err
	}

	provider, ok := s.beacon.Node().Service().(eth2client.ProposalProvider)
	if !ok {
		s.log.Error("Beacon node service client is not ProposalProvider")

		return nil, nil
	}

	// RandaoReveal must be set to the point at infinity (0xc0..00) if we're skipping Randao verification.
	rsp, err := provider.Proposal(ctx, &api.ProposalOpts{
		Slot: phase0.Slot(slot.Number()),
		RandaoReveal: phase0.BLSSignature{
			0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		},
		SkipRandaoVerification: true,
	})
	if err != nil {
		s.log.WithError(err).Error("Failed to get proposal")

		return nil, err
	}

	proposedBlock, err := getVersionedProposalData(rsp)
	if err != nil {
		return nil, err
	}

	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	snapshot.RequestDuration = time.Since(snapshot.RequestAt)

	return v2.NewValidatorBeaconBlock(s.log, proposedBlock, snapshot, s.beacon, meta), nil
}

func (s *Sentry) fetchDecoratedValidatorBeaconBlock(ctx context.Context) error {
	fc, err := s.fetchValidatorBeaconBlock(ctx)
	if err != nil {
		return err
	}

	ignore, err := fc.ShouldIgnore(ctx)
	if err != nil {
		return err
	}

	if ignore {
		return nil
	}

	decoratedEvent, err := fc.Decorate(ctx)
	if err != nil {
		return err
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func getVersionedProposalData[T any](response *api.Response[T]) (*api.VersionedProposal, error) {
	data, ok := any(response.Data).(*api.VersionedProposal)
	if !ok {
		return nil, fmt.Errorf("unexpected type for response data, expected *api.VersionedProposal, got %T", response.Data)
	}
	return data, nil
}
