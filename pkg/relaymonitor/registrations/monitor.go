package registrations

import (
	"context"
	"math"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/ethereum"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
	"github.com/go-co-op/gocron/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ValidatorRegistrationEvent struct {
	Validator    *apiv1.Validator
	Relay        *relay.Client
	Registration *mevrelay.ValidatorRegistration
}

type ValidatorMonitor struct {
	log    logrus.FieldLogger
	config *Config

	relays     []*relay.Client
	beaconNode *ethereum.BeaconNode

	walker *ValidatorSetWalker

	relayScrapers map[string]*RelayValidatorRegistrationScraper

	eventsCh chan *ValidatorRegistrationEvent
}

func NewValidatorMonitor(
	log logrus.FieldLogger,
	config *Config,
	relays []*relay.Client,
	beaconNode *ethereum.BeaconNode,
	eventsCh chan *ValidatorRegistrationEvent,
) *ValidatorMonitor {
	return &ValidatorMonitor{
		log:    log.WithField("component", "relaymonitor/validator_registration_monitor"),
		config: config,

		relays:     relays,
		beaconNode: beaconNode,

		walker: NewValidatorSetWalker(config.Shard, config.NumShards),

		relayScrapers: make(map[string]*RelayValidatorRegistrationScraper),

		eventsCh: eventsCh,
	}
}

func (r *ValidatorMonitor) Start(ctx context.Context) error {
	if r.config.Enabled == nil || !*r.config.Enabled {
		r.log.Info("Validator registration monitoring is disabled")

		return nil
	}

	if len(r.relays) == 0 {
		return errors.New("no relays found")
	}

	c, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	r.log.
		WithField("interval", r.config.ActiveValidatorsFetchInterval.String()).
		Info("Scheduling active validators fetch job")

	if _, err := c.NewJob(
		gocron.DurationJob(r.config.ActiveValidatorsFetchInterval.Duration),
		gocron.NewTask(
			func(ctx context.Context) {
				if err := r.updateActiveValidators(ctx); err != nil {
					r.log.WithError(err).Error("Failed to update active validators")
				}
			},
			ctx,
		),
	); err != nil {
		return err
	}

	// Get our first set of validators
	if err := r.updateActiveValidators(ctx); err != nil {
		return errors.Wrap(err, "failed to get active validators on startup")
	}

	callback := func(ctx context.Context, validator *apiv1.Validator, relay *relay.Client, registration *mevrelay.ValidatorRegistration) {
		r.eventsCh <- &ValidatorRegistrationEvent{
			Validator:    validator,
			Relay:        relay,
			Registration: registration,
		}
	}

	r.log.
		WithField("workers", r.config.Workers).
		WithField("num_relays", len(r.relays)).
		Info("Starting validator registration scrapers")

	for _, relay := range r.relays {
		scraper := NewRelayValidatorRegistrationScraper(r.log, r.config.Workers, relay, callback)

		if err := scraper.Start(ctx); err != nil {
			return errors.Wrap(err, "failed to start validator registration scraper")
		}

		r.relayScrapers[relay.Name()] = scraper
	}

	c.Start()

	r.startFeedingValidators(ctx)

	return nil
}

// updateActiveValidators fetches the active validators from the beacon node and updates the local cache
func (r *ValidatorMonitor) updateActiveValidators(ctx context.Context) error {
	activeValidators, err := r.beaconNode.GetActiveValidators(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get active validators")
	}

	r.walker.Update(activeValidators)

	return nil
}

func (r *ValidatorMonitor) startFeedingValidators(ctx context.Context) {
	go func() {
		validators := r.walker.NumValidatorsInShard()

		// Create a ticker that ensures we feed validators at a regular interval
		// by calculating the time it would take to feed all validators
		// to meet the target sweep duration.

		//nolint:gosec // Not a concern
		interval := r.config.TargetSweepDuration.Duration / time.Duration(validators)

		r.log.
			WithField("target_sweep_duration", r.config.TargetSweepDuration.String()).
			WithField("num_validators", validators).
			WithField("shard", r.walker.Shard()).
			WithField("num_shards", r.walker.NumShards()).
			WithField("validators_in_shard", r.walker.NumValidatorsInShard()).
			WithField("validators_per_second", math.Round(1/interval.Seconds()*100)/100).
			Info("Starting validator feeder")

		ticker := time.NewTicker(interval)

		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.log.Info("Stopping validator feeder")

				return
			case <-ticker.C:
				next, err := r.walker.Next()
				if err != nil {
					r.log.WithError(err).Error("Failed to get next validator index")

					continue
				}

				validator, err := r.walker.GetValidator(&next)
				if err != nil {
					r.log.WithError(err).Error("Failed to get validator")

					continue
				}

				for _, scraper := range r.relayScrapers {
					scraper.Add(validator)
				}
			}
		}
	}()
}
