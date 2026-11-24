package sentry

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/ethwallclock"
	eth2client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/go-co-op/gocron/v2"

	v3 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v3"
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

					slot, _, err := s.beacon.Metadata().Wallclock().Now()
					if err != nil {
						logCtx.WithError(err).Error("Failed to get current slot")

						return
					}

					if err := s.fetchDecoratedValidatorBlock(ctx, phase0.Slot(slot.Number())); err != nil {
						logCtx.WithError(err).Error("Failed to fetch validator beacon block")
					}
				},
				ctx,
			),
			gocron.WithStartAt(gocron.WithStartImmediately()),
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

		if err := s.fetchDecoratedValidatorBlock(ctx, phase0.Slot(slot.Number())); err != nil {
			logCtx.WithField("slot_time", offset.String()).WithError(err).Error("Failed to fetch validator beacon block")
		}
	})
}

// infinityRandaoReveal is the BLS point at infinity, the randao reveal beacon
// nodes expect when randao verification is skipped.
var infinityRandaoReveal = phase0.BLSSignature{0xc0}

func (s *Sentry) fetchValidatorBlock(ctx context.Context, slot phase0.Slot) (*v3.ValidatorBlock, error) {
	snapshot := &v3.ValidatorBlockDataSnapshot{RequestAt: time.Now()}

	postGloas, err := s.gloasActiveAt(slot)
	if err != nil {
		return nil, err
	}

	var (
		proposedBlock *api.VersionedProposal
		envelope      *gloas.ExecutionPayloadEnvelope
	)

	if postGloas {
		proposedBlock, envelope, err = s.produceBlockV4(ctx, slot, &infinityRandaoReveal)
	} else {
		proposedBlock, err = s.produceBlockV3(ctx, slot)
	}

	if err != nil {
		s.log.WithError(err).WithContext(ctx).Error("Failed to get proposal")

		return nil, err
	}

	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	snapshot.RequestDuration = time.Since(snapshot.RequestAt)

	return v3.NewValidatorBlock(s.log, proposedBlock, envelope, snapshot, s.beacon, meta), nil
}

func (s *Sentry) produceBlockV3(ctx context.Context, slot phase0.Slot) (*api.VersionedProposal, error) {
	provider, ok := s.beacon.Node().Service().(eth2client.ProposalProvider)
	if !ok {
		return nil, fmt.Errorf("unexpected service client type, expected: eth2client.ProposalProvider, got %T", s.beacon.Node().Service())
	}

	// Percentage multiplier to apply to the builder's payload value when choosing between a builder payload header
	// and payload from the paired execution node.
	// See https://ethereum.github.io/beacon-APIs/#/Validator/produceBlockV3
	boostFactor := uint64(0)

	rsp, err := provider.Proposal(ctx, &api.ProposalOpts{
		Slot:                   slot,
		RandaoReveal:           infinityRandaoReveal,
		SkipRandaoVerification: true,
		BuilderBoostFactor:     &boostFactor,
	})
	if err != nil {
		return nil, err
	}

	return getVersionedProposalData(rsp)
}

func (s *Sentry) gloasActiveAt(slot phase0.Slot) (bool, error) {
	sp, err := s.beacon.Node().Spec()
	if err != nil {
		return false, fmt.Errorf("failed to get spec: %w", err)
	}

	fork, err := sp.ForkEpochs.CurrentFork(phase0.Epoch(uint64(slot) / uint64(sp.SlotsPerEpoch)))
	if err != nil {
		return false, fmt.Errorf("failed to get fork at slot %d: %w", slot, err)
	}

	return fork.Name >= spec.DataVersionGloas, nil
}

func (s *Sentry) fetchDecoratedValidatorBlock(ctx context.Context, slot phase0.Slot) error {
	fc, err := s.fetchValidatorBlock(ctx, slot)
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
