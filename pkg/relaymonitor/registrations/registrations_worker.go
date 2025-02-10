package registrations

import (
	"context"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
	"github.com/sirupsen/logrus"
)

type RelayValidatorRegistrationScraper struct {
	log             logrus.FieldLogger
	workers         int
	relay           *relay.Client
	pending         chan *apiv1.Validator
	rateLimitedTill time.Time
	cb              func(ctx context.Context, validator *apiv1.Validator, relay *relay.Client, registration *mevrelay.ValidatorRegistration)
}

// NewRelayValidatorRegistrationScraper creates a new validator registration scraper
func NewRelayValidatorRegistrationScraper(
	log logrus.FieldLogger,
	workers int,
	r *relay.Client,
	cb func(ctx context.Context, validator *apiv1.Validator, relay *relay.Client, registration *mevrelay.ValidatorRegistration),
) *RelayValidatorRegistrationScraper {
	return &RelayValidatorRegistrationScraper{
		log: log.
			WithField("component", "relaymonitor/relay_validator_registration_scraper").
			WithField("relay", r.Name()),
		workers: workers,
		relay:   r,
		pending: make(chan *apiv1.Validator, 10000),
		cb:      cb,
	}
}

func (w *RelayValidatorRegistrationScraper) Start(ctx context.Context) error {
	w.log.WithField("workers", w.workers).Info("Starting validator registration scraper")

	for i := 0; i < w.workers; i++ {
		go w.startWorker(ctx, i)
	}

	return nil
}

func (w *RelayValidatorRegistrationScraper) Stop() {
	close(w.pending)
}

func (w *RelayValidatorRegistrationScraper) startWorker(ctx context.Context, _ int) {
	for {
		select {
		case <-ctx.Done():
			close(w.pending)

			return
		case validator, ok := <-w.pending:
			if !ok {
				return
			}

			if validator == nil {
				w.log.Warn("Received nil validator, skipping")

				continue
			}

			registration, err := w.relay.GetValidatorRegistrations(ctx, validator.Validator.PublicKey.String())
			if err != nil {
				if err == relay.ErrValidatorNotRegistered {
					continue
				}

				if err == relay.ErrRateLimited {
					w.rateLimitedTill = time.Now().Add(time.Second * 30)

					w.log.
						WithField("rate_limited_till", w.rateLimitedTill).
						Warn("Received rate limited response from relay, backing off. If this persists, adjust targetSweepDuration in the config.")

					continue
				}

				w.log.WithError(err).WithField("pubkey", validator.Validator.PublicKey.String()).Error("Failed to get validator registration")

				continue
			}

			w.cb(ctx, validator, w.relay, registration)
		}
	}
}

func (w *RelayValidatorRegistrationScraper) Add(validator *apiv1.Validator) {
	// If we're full, don't add more validators
	// We don't want to block the validator feeder if we're rate limited.
	// Or the relay can't handle the load.
	if len(w.pending) >= cap(w.pending) {
		return
	}

	// If we're rate limited, don't add more validators
	if time.Now().Before(w.rateLimitedTill) {
		return
	}

	w.pending <- validator
}
