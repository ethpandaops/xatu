package server

import (
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/service"
	"github.com/ethpandaops/xatu/pkg/server/store"
)

type Config struct {
	// The address to listen on.
	Addr string `yaml:"addr" default:":8080"`
	// PreStopSleepSeconds is the number of seconds to sleep before stopping.
	// Useful for giving kubernetes time to drain connections.
	// This sleep will happen after a SIGTERM is received, and will
	// delay the shutdown of the server and all of it's components.
	// Note: Do not set this to a value greater than the kubernetes
	// terminationGracePeriodSeconds.
	PreStopSleepSeconds int `yaml:"preStopSleepSeconds" default:"0"`
	// MetricsAddr is the address to listen on for metrics.
	MetricsAddr string `yaml:"metricsAddr" default:":9090"`
	// PProfAddr is the address to listen on for pprof.
	PProfAddr *string `yaml:"pprofAddr"`
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging" default:"info"`

	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`

	// Store is the cache configuration.
	Persistence persistence.Config `yaml:"persistence"`
	// Store is the cache configuration.
	Store store.Config `yaml:"store"`
	// GeoIP is the geoip provider configuration.
	GeoIP geoip.Config `yaml:"geoip"`

	// Services is the list of services to run.
	Services service.Config `yaml:"services"`

	// Tracing configuration
	Tracing observability.TracingConfig `yaml:"tracing"`
}

func (c *Config) Validate() error {
	if err := c.Services.Validate(); err != nil {
		return err
	}

	if err := c.Persistence.Validate(); err != nil {
		return err
	}

	if err := c.Store.Validate(); err != nil {
		return err
	}

	if err := c.GeoIP.Validate(); err != nil {
		return err
	}

	if err := c.Tracing.Validate(); err != nil {
		return err
	}

	return nil
}
