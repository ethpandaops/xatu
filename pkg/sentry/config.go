package sentry

import (
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/sirupsen/logrus"
)

type Config struct {
	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`

	// The name of the sentry
	Name string `yaml:"name"`

	// Ethereum configuration
	Ethereum ethereum.Config `yaml:"ethereum"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`

	// Labels configures the sentry with labels
	Labels map[string]string `yaml:"labels"`

	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`
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
