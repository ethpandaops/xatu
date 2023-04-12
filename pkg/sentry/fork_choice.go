package sentry

import (
	"context"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	v1 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v1"
)

func (s *Sentry) startForkChoiceSchedule(ctx context.Context) error {
	if !s.Config.ForkChoice.Enabled {
		return nil
	}

	if s.Config.ForkChoice.OnReOrgEvent.Enabled {
		logCtx := s.log.WithField("proccer", "fork_choice_event")

		s.beacon.Node().OnChainReOrg(ctx, func(ctx context.Context, chainReorg *eth2v1.ChainReorgEvent) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				logCtx.WithError(err).Error("Failed to create client meta when fetching fork choice after re-org event")

				return err
			}

			// Store the latest fork choice so that we can use it in the re-org event.
			// We need to store it since fetchDebugForkChoice() will update the latest fork choice
			// after it fetches the new one.
			var latestForkChoice v1.ForkChoice
			validLatestForkChoice := false

			if s.latestForkChoice != nil {
				latestForkChoice = *s.latestForkChoice
				validLatestForkChoice = true
			}

			after, err := s.fetchDebugForkChoice(ctx)
			if err != nil {
				logCtx.WithError(err).Error("Failed to fetch fork choice after re-org event")

				return err
			}

			// Lock the latest fork choice so that we can't update it while we're
			// processing the re-org event.
			s.latestForkChoiceMu.Lock()
			defer s.latestForkChoiceMu.Unlock()

			snapshot := &v1.ForkChoiceReOrgSnapshot{
				Before:       nil, // May be nil
				After:        after,
				ReOrgEventAt: now,
			}

			if validLatestForkChoice {
				snapshot.Before = &latestForkChoice
			}

			debugForkChoiceReOrgEvent := v1.NewForkChoiceReOrg(s.log, snapshot, s.beacon, meta)

			decoratedEvent, err := debugForkChoiceReOrgEvent.Decorate(ctx)
			if err != nil {
				logCtx.WithError(err).Error("Failed to decorate fork choice re-org event")

				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})
	}

	if s.Config.ForkChoice.Interval.Enabled {
		logCtx := s.log.WithField("proccer", "interval").WithField("interval", s.Config.ForkChoice.Interval.Every.String())

		if _, err := s.scheduler.Every(s.Config.ForkChoice.Interval.Every.Duration).Do(func() {
			logCtx.Debug("Fetching debug fork choice")

			err := s.fetchDecoratedDebugForkChoice(ctx)
			if err != nil {
				logCtx.WithError(err).Error("Failed to fetch debug fork choice")
			}
		}); err != nil {
			return err
		}
	}

	if s.Config.ForkChoice.At.Enabled {
		for _, slotTime := range s.Config.ForkChoice.At.SlotTimes {
			s.scheduleForkChoiceFetchingAtSlotTime(ctx, slotTime.Duration)
		}
	}

	return nil
}

func (s *Sentry) scheduleForkChoiceFetchingAtSlotTime(ctx context.Context, at time.Duration) {
	offset := at

	logCtx := s.log.
		WithField("proccer", "at_slot_time").
		WithField("slot_time", offset.String())

	s.beacon.Metadata().Wallclock().OnSlotChanged(func(slot ethwallclock.Slot) {
		time.Sleep(offset)

		logCtx.WithField("slot", slot.Number()).Debug("Fetching debug fork choice")

		err := s.fetchDecoratedDebugForkChoice(ctx)
		if err != nil {
			logCtx.WithField("slot_time", offset.String()).WithError(err).Error("Failed to fetch debug fork choice")
		}
	})
}

func (s *Sentry) fetchDebugForkChoice(ctx context.Context) (*v1.ForkChoice, error) {
	startedAt := time.Now()

	slot, epoch, err := s.beacon.Metadata().Wallclock().Now()
	if err != nil {
		return nil, err
	}

	snapshot := &v1.ForkChoiceSnapshot{
		RequestAt:    startedAt,
		RequestSlot:  phase0.Slot(slot.Number()),
		RequestEpoch: phase0.Epoch(epoch.Number()),
	}

	forkChoice, err := s.beacon.Node().FetchForkChoice(ctx)
	if err != nil {
		return nil, err
	}

	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	snapshot.RequestDuration = time.Since(startedAt)
	snapshot.Event = forkChoice

	fc := v1.NewForkChoice(s.log, snapshot, s.beacon, meta)

	s.latestForkChoiceMu.Lock()
	defer s.latestForkChoiceMu.Unlock()

	s.latestForkChoice = fc

	return fc, nil
}

func (s *Sentry) fetchDecoratedDebugForkChoice(ctx context.Context) error {
	fc, err := s.fetchDebugForkChoice(ctx)
	if err != nil {
		return err
	}

	decoratedEvent, err := fc.Decorate(ctx)
	if err != nil {
		return err
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}
