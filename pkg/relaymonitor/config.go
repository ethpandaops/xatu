package relaymonitor

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/coordinator"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/ethereum"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/registrations"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
)

type Config struct {
	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`
	ProbeAddr    *string `yaml:"probeAddr"`

	// The name of the mimicry
	Name string `yaml:"name"`

	// Ethereum configuration
	Ethereum ethereum.Config `yaml:"ethereum"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`

	// Labels configures the mimicry with labels
	Labels map[string]string `yaml:"labels"`

	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`

	Schedule Schedule `yaml:"schedule"`

	Relays []relay.Config `yaml:"relays"`

	FetchProposerPayloadDelivered bool `yaml:"fetchProposerPayloadDelivered" default:"true"`

	ValidatorRegistrations registrations.Config `yaml:"validatorRegistrations"`

	// Coordinator configuration for persistence
	Coordinator *coordinator.Config `yaml:"coordinator"`

	// Consistency configuration for ensuring complete slot data
	Consistency *ConsistencyConfig `yaml:"consistency"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return fmt.Errorf("invalid ethereum config: %w", err)
	}

	if err := c.Schedule.Validate(); err != nil {
		return fmt.Errorf("invalid schedule config: %w", err)
	}

	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("invalid output config %s: %w", output.Name, err)
		}
	}

	for _, relay := range c.Relays {
		if err := relay.Validate(); err != nil {
			return fmt.Errorf("invalid relay config %s: %w", relay.Name, err)
		}
	}

	if err := c.ValidatorRegistrations.Validate(); err != nil {
		return fmt.Errorf("invalid validator registrations config: %w", err)
	}

	if c.Consistency != nil {
		if err := c.Consistency.Validate(); err != nil {
			return fmt.Errorf("invalid consistency config: %w", err)
		}
	}

	return nil
}

type Schedule struct {
	AtSlotTimes []human.Duration `yaml:"atSlotTimes"`
}

func (s *Schedule) Validate() error {
	if len(s.AtSlotTimes) == 0 {
		return errors.New("atSlotTimes must be provided")
	}

	return nil
}

// ConsistencyConfig configures background processes to ensure complete slot data
type ConsistencyConfig struct {
	// CheckEveryDuration is how often to check for new work
	CheckEveryDuration human.Duration `yaml:"checkEveryDuration" default:"1s"`

	// RateLimitPerRelay is the maximum requests per second per relay
	// Supports fractional rates (e.g., 0.5 = 1 request every 2 seconds)
	// Both forward fill and backfill share this limit, with forward fill getting priority
	RateLimitPerRelay float64 `yaml:"rateLimitPerRelay" default:"0.5"`

	// Backfill configuration for historical data
	Backfill *BackfillConfig `yaml:"backfill"`

	// ForwardFill configuration for catching up gaps
	ForwardFill *ForwardFillConfig `yaml:"forwardFill"`
}

func (c *ConsistencyConfig) Validate() error {
	// Ensure we have a valid check interval
	if c.CheckEveryDuration.Duration <= 0 {
		return errors.New("checkEveryDuration must be positive")
	}

	if c.RateLimitPerRelay <= 0 {
		return errors.New("rateLimitPerRelay must be positive (supports decimals, e.g., 0.5 for 1 req/2s)")
	}

	// Validate ForwardFill configuration if present
	if c.ForwardFill != nil && c.ForwardFill.Enabled {
		if c.ForwardFill.TrailDistance == 0 {
			return errors.New("forwardFill.trailDistance must be greater than 0 when forward fill is enabled")
		}

		if c.ForwardFill.TrailDistance > 1000 {
			return errors.New("forwardFill.trailDistance exceeds maximum allowed value of 1000 slots (~3.3 hours)")
		}
	}

	return nil
}

// BackfillConfig configures historical slot data backfilling
type BackfillConfig struct {
	// Enabled controls whether backfill is active
	Enabled bool `yaml:"enabled" default:"false"`

	// ToSlot is the minimum slot to backfill to (0 for genesis)
	ToSlot uint64 `yaml:"toSlot" default:"0"`
}

// ForwardFillConfig configures forward filling to catch up gaps
type ForwardFillConfig struct {
	// Enabled controls whether forward fill is active
	Enabled bool `yaml:"enabled" default:"false"`

	// TrailDistance is how many slots behind the head slot we will trail
	// This prevents attempting to fetch data for slots that may not yet be available
	TrailDistance uint64 `yaml:"trailDistance" default:"32"`
}

func (c *Config) CreateSinks(log logrus.FieldLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodAsync
			out.ShippingMethod = &shippingMethod
		}

		sink, err := output.NewSink(
			out.Name,
			out.SinkType,
			out.Config,
			log,
			out.FilterConfig,
			*out.ShippingMethod,
		)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}

// ApplyOverrides applies any overrides to the config.
func (c *Config) ApplyOverrides(o *Override, log logrus.FieldLogger) error {
	if o == nil {
		return nil
	}

	if o.MetricsAddr.Enabled {
		log.WithField("address", o.MetricsAddr.Value).Info("Overriding metrics address")

		c.MetricsAddr = o.MetricsAddr.Value
	}

	return nil
}
