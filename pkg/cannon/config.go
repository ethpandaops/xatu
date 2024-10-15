package cannon

import (
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/deriver"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/sirupsen/logrus"
)

type Config struct {
	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`

	// The name of the cannon
	Name string `yaml:"name"`

	// Ethereum configuration
	Ethereum ethereum.Config `yaml:"ethereum"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`

	// Labels configures the cannon with labels
	Labels map[string]string `yaml:"labels"`

	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`

	// Derivers configures the cannon with event derivers
	Derivers deriver.Config `yaml:"derivers"`

	// Coordinator configuration
	Coordinator coordinator.Config `yaml:"coordinator"`

	// Tracing configuration
	Tracing observability.TracingConfig `yaml:"tracing"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return err
	}

	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("invalid output config %s: %w", output.Name, err)
		}
	}

	if err := c.Derivers.Validate(); err != nil {
		return fmt.Errorf("invalid derivers config: %w", err)
	}

	if err := c.Coordinator.Validate(); err != nil {
		return fmt.Errorf("invalid coordinator config: %w", err)
	}

	if err := c.Tracing.Validate(); err != nil {
		return fmt.Errorf("invalid tracing config: %w", err)
	}

	return nil
}

func (c *Config) CreateSinks(log logrus.FieldLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodSync

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
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}

func (c *Config) ApplyOverrides(o *Override, log logrus.FieldLogger) error {
	if o == nil {
		return nil
	}

	if o.BeaconNodeURL.Enabled {
		log.Info("Overriding beacon node URL")

		c.Ethereum.BeaconNodeAddress = o.BeaconNodeURL.Value
	}

	if o.BeaconNodeAuthorizationHeader.Enabled {
		log.Info("Overriding beacon node authorization header")

		c.Ethereum.BeaconNodeHeaders["Authorization"] = o.BeaconNodeAuthorizationHeader.Value
	}

	if o.XatuCoordinatorAuth.Enabled {
		log.Info("Overriding xatu coordinator authorization")

		c.Coordinator.Headers["Authorization"] = o.XatuCoordinatorAuth.Value
	}

	return nil
}
