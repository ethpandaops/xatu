package ethstats

import (
	"fmt"
	"time"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/auth"
	"github.com/ethpandaops/xatu/pkg/geoip"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/sirupsen/logrus"
)

// Config holds the configuration for the ethstats service.
type Config struct {
	// LoggingLevel is the logging level for the service.
	LoggingLevel string `yaml:"logging" default:"info"`
	// MetricsAddr is the address to listen on for metrics.
	MetricsAddr string `yaml:"metricsAddr" default:":9090"`
	// PProfAddr is the address to listen on for pprof. If not set, pprof is disabled.
	PProfAddr *string `yaml:"pprofAddr"`

	// Name is the name of this ethstats instance.
	Name string `yaml:"name"`

	// Addr is the address to listen on for WebSocket connections.
	Addr string `yaml:"addr" default:":3000"`

	// WebSocket contains WebSocket server configuration.
	WebSocket WebSocketConfig `yaml:"websocket"`

	// Authorization contains the authentication and authorization configuration.
	Authorization auth.AuthorizationConfig `yaml:"authorization"`

	// ClientNameSalt is the salt used for obscuring client names.
	ClientNameSalt string `yaml:"clientNameSalt"`

	// Outputs contains the output sink configurations.
	Outputs []output.Config `yaml:"outputs"`

	// Labels contains additional labels for this instance.
	Labels map[string]string `yaml:"labels"`

	// NTPServer is the NTP server to use for clock drift correction.
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`

	// Tracing contains the tracing configuration.
	Tracing observability.TracingConfig `yaml:"tracing"`

	// GeoIP contains the GeoIP configuration.
	GeoIP geoip.Config `yaml:"geoip"`

	// OverrideNetworkName overrides the network name for all events.
	// Useful for testnets where the network name should be different from what clients report.
	OverrideNetworkName string `yaml:"overrideNetworkName"`
}

// WebSocketConfig contains WebSocket server configuration.
type WebSocketConfig struct {
	// ReadBufferSize is the buffer size for reading WebSocket messages.
	ReadBufferSize int `yaml:"readBufferSize" default:"1024"`
	// WriteBufferSize is the buffer size for writing WebSocket messages.
	WriteBufferSize int `yaml:"writeBufferSize" default:"1024"`
	// PingInterval is the interval between ping messages.
	PingInterval time.Duration `yaml:"pingInterval" default:"15s"`
	// ReadLimit is the maximum message size in bytes.
	ReadLimit int64 `yaml:"readLimit" default:"15728640"`
	// WriteWait is the write deadline duration.
	WriteWait time.Duration `yaml:"writeWait" default:"10s"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if c.Addr == "" {
		return fmt.Errorf("addr is required")
	}

	if len(c.Outputs) == 0 {
		return fmt.Errorf("at least one output is required")
	}

	for _, out := range c.Outputs {
		if err := out.Validate(); err != nil {
			return fmt.Errorf("output %s: %w", out.Name, err)
		}
	}

	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization: %w", err)
	}

	if err := c.Tracing.Validate(); err != nil {
		return fmt.Errorf("tracing: %w", err)
	}

	if err := c.WebSocket.Validate(); err != nil {
		return fmt.Errorf("websocket: %w", err)
	}

	return nil
}

// Validate validates the WebSocket configuration.
func (c *WebSocketConfig) Validate() error {
	if c.ReadBufferSize <= 0 {
		return fmt.Errorf("readBufferSize must be positive")
	}

	if c.WriteBufferSize <= 0 {
		return fmt.Errorf("writeBufferSize must be positive")
	}

	if c.PingInterval <= 0 {
		return fmt.Errorf("pingInterval must be positive")
	}

	if c.ReadLimit <= 0 {
		return fmt.Errorf("readLimit must be positive")
	}

	if c.WriteWait <= 0 {
		return fmt.Errorf("writeWait must be positive")
	}

	return nil
}

// CreateSinks creates output sinks from the configuration.
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
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}

// ApplyDefaults applies default values to the configuration.
func (c *Config) ApplyDefaults() error {
	return defaults.Set(c)
}

// ApplyOverrides applies runtime overrides to the configuration.
func (c *Config) ApplyOverrides(o *Override, log logrus.FieldLogger) error {
	if o == nil {
		return nil
	}

	if o.MetricsAddr.Enabled {
		log.WithField("address", o.MetricsAddr.Value).Info("Overriding metrics address")

		c.MetricsAddr = o.MetricsAddr.Value
	}

	if o.NetworkName.Enabled {
		log.WithField("network", o.NetworkName.Value).Info("Overriding network name")

		c.OverrideNetworkName = o.NetworkName.Value
	}

	return nil
}
