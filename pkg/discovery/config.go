package discovery

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/discovery/coordinator"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p"
)

type Config struct {
	LoggingLevel string `yaml:"logging" default:"info"`
	MetricsAddr  string `yaml:"metricsAddr" default:":9090"`

	// P2P configuration
	P2P p2p.Config `yaml:"p2p"`

	// Coordinator configuration
	Coordinator coordinator.Config `yaml:"coordinator"`
}

func (c *Config) Validate() error {
	if err := c.P2P.Validate(); err != nil {
		return fmt.Errorf("p2p config error: %w", err)
	}

	if err := c.Coordinator.Validate(); err != nil {
		return fmt.Errorf("coordinator config error: %w", err)
	}

	return nil
}
