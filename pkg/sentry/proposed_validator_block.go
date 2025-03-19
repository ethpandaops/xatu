package sentry

import (
	"context"
	"fmt"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	v3 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v3"
	"github.com/go-co-op/gocron/v2"
)

func (s *Sentry) startValidatorBlockSchedule(ctx context.Context) error {
	if !s.Config.ValidatorBlock.Enabled {
		return nil
	}

	if s.Config.ValidatorBlock.Interval.Enabled {
		logCtx := s.log.WithField("proccer", "interval").WithField("interval", s.Config.ValidatorBlock.Interval.Every.String())

		if _, err := s.scheduler.NewJob(
			gocron.DurationJob(s.Config.ValidatorBlock.Interval.Every.Duration),
			gocron.NewTask(
				func(ctx context.Context) {
					logCtx.Debug("Fetching validator beacon block")

					if err := s.fetchDecoratedValidatorBlock(ctx); err != nil {
						logCtx.WithError(err).Error("Failed to fetch validator beacon block")
					}
				},
				ctx,
			),
		); err != nil {
			return err
		}
	}

	if s.Config.ValidatorBlock.At.Enabled {
		for _, slotTime := range s.Config.ValidatorBlock.At.SlotTimes {
			s.scheduleValidatorBlockFetchingAtSlotTime(ctx, slotTime.Duration)
		}
	}

	return nil
}

func (s *Sentry) scheduleValidatorBlockFetchingAtSlotTime(ctx context.Context, at time.Duration) {
	offset := at

	logCtx := s.log.
		WithField("proccer", "at_slot_time").
		WithField("slot_time", offset.String())

	logCtx.Debug("Scheduling validator beacon block fetching at slot time")

	s.beacon.Metadata().Wallclock().OnSlotChanged(func(slot ethwallclock.Slot) {
		time.Sleep(offset)

		logCtx.WithField("slot", slot.Number()).Debug("Fetching validator beacon block")

		if err := s.fetchDecoratedValidatorBlock(ctx); err != nil {
			logCtx.WithField("slot_time", offset.String()).WithError(err).Error("Failed to fetch validator beacon block")
		}
	})
}

func (s *Sentry) fetchValidatorBlock(ctx context.Context) (*v3.ValidatorBlock, error) {
	snapshot := &v3.ValidatorBlockDataSnapshot{RequestAt: time.Now()}

	slot, _, err := s.beacon.Metadata().Wallclock().Now()
	if err != nil {
		return nil, err
	}

	provider, ok := s.beacon.Node().Service().(eth2client.ProposalProvider)
	if !ok {
		s.log.Error("Beacon node service client is not ProposalProvider")

		return nil, fmt.Errorf("unexpected service client type, expected: eth2client.ProposalProvider, got %T", s.beacon.Node().Service())
	}

	// Percentage multiplier to apply to the builder's payload value when choosing between a builder payload header
	// and payload from the paired execution node.
	// See https://ethereum.github.io/beacon-APIs/#/Validator/produceBlockV3
	boostFactor := uint64(0)

	// RandaoReveal must be set to the point at infinity (0xc0..00) if we're skipping Randao verification.
	rsp, err := provider.Proposal(ctx, &api.ProposalOpts{
		Slot: phase0.Slot(slot.Number() + 1),
		RandaoReveal: phase0.BLSSignature{
			0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		},
		SkipRandaoVerification: true,
		BuilderBoostFactor:     &boostFactor,
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

	return v3.NewValidatorBlock(s.log, proposedBlock, snapshot, s.beacon, meta), nil
}

func (s *Sentry) fetchDecoratedValidatorBlock(ctx context.Context) error {
	fc, err := s.fetchValidatorBlock(ctx)
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
