package relaymonitor

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
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

	// Backfill configuration for historical slot data
	Backfill *BackfillConfig `yaml:"backfill"`
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

	if c.Backfill != nil {
		if err := c.Backfill.Validate(); err != nil {
			return fmt.Errorf("invalid backfill config: %w", err)
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

// BackfillConfig configures historical slot data backfilling
type BackfillConfig struct {
	// Enabled controls whether backfill is active
	Enabled bool `yaml:"enabled" default:"false"`

	// To configures how far back to backfill
	To BackfillTo `yaml:"to"`

	// RateLimit controls backfill speed
	RateLimit BackfillRateLimit `yaml:"rateLimit"`
}

// BackfillTo specifies the backfill target
type BackfillTo struct {
	// Epoch number to backfill to (-1 for genesis)
	Epoch *int64 `yaml:"epoch"`

	// Fork name to backfill to (e.g., "bellatrix", "capella", "deneb")
	Fork *string `yaml:"fork"`
}

// BackfillRateLimit controls backfill request rate
type BackfillRateLimit struct {
	// RequestsPerSecond limits API requests per second per relay
	RequestsPerSecond int `yaml:"requestsPerSecond" default:"2"`

	// SlotsPerRequest controls how many slots to fetch per request
	SlotsPerRequest int `yaml:"slotsPerRequest" default:"10"`

	// DelayBetweenRelays adds delay between different relay requests (in milliseconds)
	DelayBetweenRelays human.Duration `yaml:"delayBetweenRelays" default:"100ms"`
}

func (c *BackfillConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// Validate target configuration
	if c.To.Epoch == nil && c.To.Fork == nil {
		// Default to genesis if nothing specified
		return nil
	}

	if c.To.Epoch != nil && c.To.Fork != nil {
		return errors.New("cannot specify both epoch and fork for backfill target")
	}

	if c.RateLimit.RequestsPerSecond <= 0 {
		return errors.New("requestsPerSecond must be greater than 0")
	}

	if c.RateLimit.SlotsPerRequest <= 0 {
		return errors.New("slotsPerRequest must be greater than 0")
	}

	return nil
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
