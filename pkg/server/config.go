package server

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/service"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
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

func (c *Config) ApplyOverrides(o *Override, log logrus.FieldLogger) error {
	if o == nil {
		return nil
	}

	if o.EventIngesterBasicAuth.Username != "" || o.EventIngesterBasicAuth.Password != "" {
		log.Info("Enabling event ingester basic authentication via override")

		if o.EventIngesterBasicAuth.Password == "" {
			return fmt.Errorf("invalid basic auth override configuration: password is required")
		}

		if o.EventIngesterBasicAuth.Username == "" {
			return fmt.Errorf("invalid basic auth override configuration: username is required")
		}

		// Event Ingester
		c.Services.EventIngester.Authorization.Enabled = true

		c.Services.EventIngester.Authorization.Groups = make(auth.GroupsConfig)
		groupName := "simple"

		c.Services.EventIngester.Authorization.Groups[groupName] = auth.GroupConfig{
			Users: auth.UsersConfig{},
		}

		c.Services.EventIngester.Authorization.Groups[groupName].Users[o.EventIngesterBasicAuth.Username] = auth.UserConfig{
			Password: o.EventIngesterBasicAuth.Password,
		}
	}

	if o.CoordinatorAuth.Enabled {
		log.Info("Enabling coordinator authentication via override")

		// Coordinator
		enabled := true
		c.Services.Coordinator.Auth.Enabled = &enabled

		c.Services.Coordinator.Auth.Secret = o.CoordinatorAuth.AuthSecret
	}

	return nil
}
