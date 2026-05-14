package cannon

import (
	"errors"
	"fmt"

	"github.com/creasty/defaults"

	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/deriver"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	chSink "github.com/ethpandaops/xatu/pkg/output/clickhouse"
	"github.com/ethpandaops/xatu/pkg/processor"
)

// cannonClickhouseMetricsSubsystem is the Prometheus subsystem cannon
// uses for the direct-CH sink. Cannon owns this — the sink package is
// binary-agnostic.
const cannonClickhouseMetricsSubsystem = "cannon"

// cannonClickhouseTablePrefix is the route-catalog prefix cannon emits.
// Cannon's derivers only produce canonical_* events; restricting the
// catalog to this prefix avoids requiring schemas the cannon process
// will never write to (libp2p_, mev_, execution_, node_).
var cannonClickhouseTablePrefixes = []string{"canonical_"}

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

func (c *Config) CreateSinks(log observability.ContextualLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodSync

			out.ShippingMethod = &shippingMethod
		}

		sink, err := c.createSink(&out, log)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}

// createSink dispatches sink construction. The clickhouse sink gets
// cannon-specific defaults applied (metrics subsystem, route-prefix
// restriction) so user YAML can stay minimal — cannon knows it is
// cannon and that its derivers only emit canonical_* events; the user
// shouldn't have to express either fact in config. All other sink
// types pass through to output.NewSink unchanged.
func (c *Config) createSink(out *output.Config, log observability.ContextualLogger) (output.Sink, error) {
	if out.SinkType == output.SinkTypeClickhouse {
		return newCannonClickhouseSink(out, log)
	}

	return output.NewSink(
		out.Name,
		out.SinkType,
		out.Config,
		log,
		out.FilterConfig,
		*out.ShippingMethod,
	)
}

// newCannonClickhouseSink mirrors the unmarshal+defaults flow that
// output.NewSink does internally for the clickhouse sink, but inserts
// cannon-specific defaults between the YAML decode and the sink
// constructor. User-supplied values always win.
func newCannonClickhouseSink(out *output.Config, log observability.ContextualLogger) (output.Sink, error) {
	conf := &chSink.Config{}

	if out.Config != nil {
		if err := out.Config.Unmarshal(conf); err != nil {
			return nil, fmt.Errorf("decoding clickhouse sink config: %w", err)
		}
	}

	if err := defaults.Set(conf); err != nil {
		return nil, fmt.Errorf("applying clickhouse sink defaults: %w", err)
	}

	if conf.MetricsSubsystem == "" {
		conf.MetricsSubsystem = cannonClickhouseMetricsSubsystem
	}

	if len(conf.RestrictToTablePrefixes) == 0 {
		conf.RestrictToTablePrefixes = append([]string(nil), cannonClickhouseTablePrefixes...)
	}

	return chSink.New(out.Name, conf, log, &out.FilterConfig, *out.ShippingMethod)
}

func (c *Config) ApplyOverrides(o *Override, log observability.ContextualLogger) error {
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

	if o.NetworkName.Enabled {
		log.WithField("network", o.NetworkName.Value).Info("Overriding network name")

		c.Ethereum.OverrideNetworkName = o.NetworkName.Value
	}

	if o.MetricsAddr.Enabled {
		log.WithField("address", o.MetricsAddr.Value).Info("Overriding metrics address")

		c.MetricsAddr = o.MetricsAddr.Value
	}

	return nil
}
