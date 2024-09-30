package eventingester

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
)

type Config struct {
	Enabled bool `yaml:"enabled" default:"false"`
	// Outputs is the list of sinks to use.
	Outputs []output.Config `yaml:"outputs"`
	// Authorization is the authorization configuration.
	Authorization auth.AuthorizationConfig `yaml:"authorization"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if len(c.Outputs) == 0 {
		return fmt.Errorf("no outputs configured")
	}

	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization config is invalid: %w", err)
	}

	return nil
}
