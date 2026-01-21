package horizon

import (
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/horizon/cache"
	"github.com/ethpandaops/xatu/pkg/horizon/coordinator"
	"github.com/ethpandaops/xatu/pkg/horizon/deriver"
	"github.com/ethpandaops/xatu/pkg/horizon/ethereum"
	"github.com/ethpandaops/xatu/pkg/horizon/iterator"
	"github.com/ethpandaops/xatu/pkg/horizon/subscription"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/sirupsen/logrus"
)

type Config struct {
	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`

	// The name of the horizon instance
	Name string `yaml:"name"`

	// Ethereum configuration (beacon node pool)
	Ethereum ethereum.Config `yaml:"ethereum"`

	// Coordinator configuration for tracking processing locations
	Coordinator coordinator.Config `yaml:"coordinator"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`

	// Labels configures the horizon with labels
	Labels map[string]string `yaml:"labels"`

	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`

	// Tracing configuration
	Tracing observability.TracingConfig `yaml:"tracing"`

	// Derivers configuration
	Derivers deriver.Config `yaml:"derivers"`

	// DedupCache configuration for block event deduplication
	DedupCache cache.Config `yaml:"dedupCache"`

	// Subscription configuration for SSE block events
	Subscription subscription.Config `yaml:"subscription"`

	// Reorg configuration for chain reorg handling
	Reorg subscription.ReorgConfig `yaml:"reorg"`

	// EpochIterator configuration for epoch-based derivers
	EpochIterator iterator.EpochIteratorConfig `yaml:"epochIterator"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return fmt.Errorf("invalid ethereum config: %w", err)
	}

	if err := c.Coordinator.Validate(); err != nil {
		return fmt.Errorf("invalid coordinator config: %w", err)
	}

	if len(c.Outputs) == 0 {
		return errors.New("at least one output sink is required")
	}

	for _, out := range c.Outputs {
		if err := out.Validate(); err != nil {
			return fmt.Errorf("invalid output config %s: %w", out.Name, err)
		}
	}

	if err := c.Tracing.Validate(); err != nil {
		return fmt.Errorf("invalid tracing config: %w", err)
	}

	if err := c.Derivers.Validate(); err != nil {
		return fmt.Errorf("invalid derivers config: %w", err)
	}

	if err := c.DedupCache.Validate(); err != nil {
		return fmt.Errorf("invalid dedup cache config: %w", err)
	}

	if err := c.Subscription.Validate(); err != nil {
		return fmt.Errorf("invalid subscription config: %w", err)
	}

	if err := c.Reorg.Validate(); err != nil {
		return fmt.Errorf("invalid reorg config: %w", err)
	}

	if err := c.EpochIterator.Validate(); err != nil {
		return fmt.Errorf("invalid epoch iterator config: %w", err)
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

	if o.MetricsAddr.Enabled {
		log.WithField("address", o.MetricsAddr.Value).Info("Overriding metrics address")

		c.MetricsAddr = o.MetricsAddr.Value
	}

	if o.BeaconNodeURLs.Enabled {
		log.Info("Overriding beacon node URLs")
	}

	if o.BeaconNodeHeaders.Enabled {
		log.Info("Overriding beacon node authorization headers")
	}

	if o.NetworkName.Enabled {
		log.WithField("network", o.NetworkName.Value).Info("Overriding network name")
	}

	o.ApplyBeaconNodeOverrides(&c.Ethereum)

	return nil
}
