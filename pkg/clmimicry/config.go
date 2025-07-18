package clmimicry

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/clmimicry/ethereum"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
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

	// Node is the configuration for the node
	Node NodeConfig `yaml:"node"`

	// Events is the configuration for the events
	Events EventConfig `yaml:"events"`

	// Sharding is the configuration for event sharding
	Sharding ShardingConfig `yaml:"sharding"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return fmt.Errorf("invalid ethereum config: %w", err)
	}

	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("invalid output config %s: %w", output.Name, err)
		}
	}

	if err := c.Events.Validate(); err != nil {
		return fmt.Errorf("invalid events config: %w", err)
	}

	if err := c.validateSharding(); err != nil {
		return fmt.Errorf("invalid sharding config: %w", err)
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

// validateSharding validates the sharding configuration
func (c *Config) validateSharding() error {
	// If sharding is not configured, that's fine - we'll use defaults
	if c.Sharding.Topics == nil {
		c.Sharding.Topics = make(map[string]*TopicShardingConfig)
	}

	// Validate topic configurations
	for pattern, config := range c.Sharding.Topics {
		if err := config.validate(pattern); err != nil {
			return err
		}
	}

	// Set default for no sharding key events if not specified
	if c.Sharding.NoShardingKeyEvents == nil {
		c.Sharding.NoShardingKeyEvents = &NoShardingKeyConfig{
			Enabled: true,
		}
	}

	return nil
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
