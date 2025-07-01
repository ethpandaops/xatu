package discovery

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/discovery/coordinator"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/sirupsen/logrus"
)

type Config struct {
	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`

	// Beacon node URL for consensus discovery
	BeaconNodeURL string `yaml:"beaconNodeUrl"`

	// P2P configuration
	P2P p2p.Config `yaml:"p2p"`

	// Coordinator configuration
	Coordinator coordinator.Config `yaml:"coordinator"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`
}

func (c *Config) Validate() error {
	if err := c.P2P.Validate(); err != nil {
		return fmt.Errorf("p2p config error: %w", err)
	}

	if err := c.Coordinator.Validate(); err != nil {
		return fmt.Errorf("coordinator config error: %w", err)
	}

	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("output %s: %w", output.Name, err)
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

// CreateSinks creates the sinks from the configuration.
func (c *Config) CreateSinks(log logrus.FieldLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodAsync
			out.ShippingMethod = &shippingMethod
		}

		sink, err := output.NewSink(out.Name,
			out.SinkType,
			out.Config,
			log,
			out.FilterConfig,
			*out.ShippingMethod,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create sink %s: %w", out.Name, err)
		}

		sinks[i] = sink
	}

	return sinks, nil
}
