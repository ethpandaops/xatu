package eventingester

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/output"
)

type Config struct {
	Enabled bool `yaml:"enabled" default:"false"`
	// Outputs is the list of sinks to use.
	Outputs []output.Config `yaml:"outputs"`
	// AuthorizationSecret is the secret to use for authorization.
	AuthorizationSecret string `yaml:"authorizationSecret"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if len(c.Outputs) == 0 {
		return fmt.Errorf("no outputs configured")
	}

	return nil
}
