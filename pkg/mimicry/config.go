package mimicry

import (
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/sirupsen/logrus"
)

type Config struct {
	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`

	// The name of the mimicry
	Name string `yaml:"name"`

	// Ethereum configuration
	Coordinator coordinator.Config `yaml:"coordinator"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`

	// Labels configures the mimicry with labels
	Labels map[string]string `yaml:"labels"`

	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if err := c.Coordinator.Validate(); err != nil {
		return err
	}

	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("output %s: %w", output.Name, err)
		}
	}

	return nil
}

func (c *Config) CreateSinks(log logrus.FieldLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		sink, err := output.NewSink(out.SinkType, out.Config, log, out.FilterConfig)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}
