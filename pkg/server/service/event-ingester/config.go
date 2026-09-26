package eventingester

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
)

type Config struct {
	Enabled bool `yaml:"enabled" default:"false"`
	// Outputs is the list of sinks to use.
	Outputs []output.Config `yaml:"outputs"`
	// Authorization is the authorization configuration.
	Authorization auth.AuthorizationConfig `yaml:"authorization"`
	// ClientNameSalt is the salt to use for computing client names
	ClientNameSalt string `yaml:"clientNameSalt"`
	// Mutations are applied to every ingested event before it is handed to
	// the outputs. Useful when the server runs as a relay that needs to
	// re-namespace events before forwarding them upstream.
	Mutations xatu.EventMutatorConfig `yaml:"mutations"`
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

	if err := c.Mutations.Validate(); err != nil {
		return fmt.Errorf("mutations config is invalid: %w", err)
	}

	return nil
}
