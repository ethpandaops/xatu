package sentry

import (
	"errors"

	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/ethpandaops/xatu/pkg/sentry/output"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// The name of the sentry
	Name string `yaml:"name"`

	// Ethereum configuration
	Ethereum ethereum.Config `yaml:"ethereum"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`

	// Labels configures the sentry with labels
	Labels map[string]string `yaml:"labels"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return err
	}

	// TODO: wrap these errors with the parent field name
	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) CreateSinks(log logrus.FieldLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		sink, err := output.NewSink(out.SinkType, out.Config, log)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}
