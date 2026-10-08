package sentry

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ethpandaops/ethwallclock"
	eth2client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/go-co-op/gocron/v2"
	"github.com/sirupsen/logrus"

	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
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
				Event:        xatuethv1.NewReorgEventV2FromGoEth2ClientEvent(chainReorg),
			}

			if validLatestForkChoice {
				snapshot.Before = &latestForkChoice
			}

			debugForkChoiceReOrgEvent := v1.NewForkChoiceReOrg(s.log, snapshot, s.beacon, meta)

			ignore, err := debugForkChoiceReOrgEvent.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

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

		if _, err := s.scheduler.NewJob(
			gocron.DurationJob(s.Config.ForkChoice.Interval.Every.Duration),
			gocron.NewTask(
				func(ctx context.Context) {
					logCtx.Debug("Fetching debug fork choice")

					err := s.fetchDecoratedDebugForkChoice(ctx)
					if err != nil {
						logCtx.WithError(err).Error("Failed to fetch debug fork choice")
					}
				},
				ctx,
			),
			gocron.WithStartAt(gocron.WithStartImmediately()),
		); err != nil {
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

	logCtx.Debug("Scheduling debug fork choice fetching at slot time")

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

	// A re-org event's before and after snapshots can come from different
	// endpoints when the v2 retry window opens or closes between them: v2
	// labels each block's empty/full/pending nodes, v1 does not.
	forkChoice, forkChoiceV2, err := s.fetchForkChoice(ctx)
	if err != nil {
		return nil, err
	}

	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	snapshot.RequestDuration = time.Since(startedAt)
	snapshot.Event = forkChoice
	snapshot.EventV2 = forkChoiceV2

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

// forkChoiceV2RetryInterval is how long to use the v1 fork choice endpoint
// after the beacon node showed it cannot serve v2 before trying v2 again.
const forkChoiceV2RetryInterval = time.Hour

// fetchForkChoice fetches the beacon node's fork choice, preferring the
// Gloas-aware GET /eth/v2/debug/fork_choice (one node per block root and
// payload status) and falling back to v1. Exactly one of the returned fork
// choices is set.
func (s *Sentry) fetchForkChoice(ctx context.Context) (*eth2v1.ForkChoice, *eth2v1.ForkChoiceV2, error) {
	provider, _ := s.beacon.Node().Service().(eth2client.ForkChoiceV2Provider)

	return fetchForkChoiceWithFallback(ctx, s.log, provider, s.beacon.Node().FetchForkChoice, &s.forkChoiceV2RetryAt)
}

// fetchForkChoiceWithFallback fetches the v2 fork choice from provider (if not
// nil, and not backing off), falling back to fetchV1.
//
// A node without the endpoint, or whose v2 response does not follow the spec
// (ethereum/beacon-APIs#615), cannot serve v2 until it is upgraded, so v1 is
// used for forkChoiceV2RetryInterval before v2 is tried again; that keeps such
// a node from being asked for, and logging, a failing v2 fork choice on every
// fetch. Any other v2 failure (a timeout, a 5xx) falls back to v1 for this
// fetch only.
func fetchForkChoiceWithFallback(
	ctx context.Context,
	log logrus.FieldLogger,
	provider eth2client.ForkChoiceV2Provider,
	fetchV1 func(context.Context) (*eth2v1.ForkChoice, error),
	retryAt *atomic.Int64,
) (*eth2v1.ForkChoice, *eth2v1.ForkChoiceV2, error) {
	if provider != nil && time.Now().UnixNano() >= retryAt.Load() {
		rsp, err := provider.ForkChoiceV2(ctx, &api.ForkChoiceOpts{})
		if err == nil {
			return nil, rsp.Data, nil
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, nil, ctxErr
		}

		switch {
		case isUnsupportedEndpoint(err):
			retryAt.Store(time.Now().Add(forkChoiceV2RetryInterval).UnixNano())
			log.WithError(err).Debug("Beacon node does not support the v2 fork choice endpoint, using v1")
		case errors.Is(err, eth2client.ErrInvalidResponse):
			retryAt.Store(time.Now().Add(forkChoiceV2RetryInterval).UnixNano())
			log.WithError(err).WithField("retry_in", forkChoiceV2RetryInterval).
				Warn("Beacon node's v2 fork choice does not follow the spec, using v1")
		default:
			log.WithError(err).Warn("Failed to fetch v2 fork choice, falling back to v1 for this fetch")
		}
	}

	forkChoice, err := fetchV1(ctx)
	if err != nil {
		return nil, nil, err
	}

	return forkChoice, nil, nil
}

// isUnsupportedEndpoint reports whether err is a beacon node rejecting an
// endpoint it does not implement.
func isUnsupportedEndpoint(err error) bool {
	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		return false
	}

	switch apiErr.StatusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return true
	default:
		return false
	}
}
