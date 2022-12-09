package server

import (
	"fmt"

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
	Services []service.Config `yaml:"services"`
}

func (c *Config) Validate() error {
	// Check for duplicate service names.
	names := make(map[service.ServiceType]struct{}, len(c.Services))
	for _, service := range c.Services {
		if _, ok := names[service.ServiceType]; ok {
			return fmt.Errorf("duplicate service: %s", service.ServiceType)
		}

		names[service.ServiceType] = struct{}{}
	}

	return nil
}
