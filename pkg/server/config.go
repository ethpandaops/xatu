package server

import (
	"github.com/ethpandaops/xatu/pkg/server/service"
	"github.com/ethpandaops/xatu/pkg/server/store"
)

type Config struct {
	// The address to listen on.
	Addr string `yaml:"addr" default:":8080"`
	// MetricsAddr is the address to listen on for metrics.
	MetricsAddr string `yaml:"metrics_addr" default:":9090"`
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging_level" default:"info"`
	// Services is the list of services to run.
	Services service.Config `yaml:"services"`
	// Store is the cache configuration.
	Store store.Config `yaml:"store"`
}

func (c *Config) Validate() error {
	if err := c.Services.Validate(); err != nil {
		return err
	}

	if err := c.Store.Validate(); err != nil {
		return err
	}

	return nil
}
