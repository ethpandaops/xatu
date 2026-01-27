package eventingester

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/auth"
	"github.com/ethpandaops/xatu/pkg/output"
)

type Config struct {
	Enabled bool `yaml:"enabled" default:"false"`
	// Outputs is the list of sinks to use.
	Outputs []output.Config `yaml:"outputs"`
	// Authorization is the authorization configuration.
	Authorization auth.AuthorizationConfig `yaml:"authorization"`
	// ClientNameSalt is the salt to use for computing client names
	ClientNameSalt string `yaml:"clientNameSalt"`
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

	if c.ClientNameSalt == "" {
		return fmt.Errorf("clientNameSalt is required")
	}

	return nil
}
