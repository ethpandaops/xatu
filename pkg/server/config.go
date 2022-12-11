package server

import (
	"github.com/ethpandaops/xatu/pkg/server/service"
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
}

func (c *Config) Validate() error {
	if err := c.Services.Validate(); err != nil {
		return err
	}

	return nil
}
