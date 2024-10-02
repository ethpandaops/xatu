package coordinator

import (
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
)

type Config struct {
	Enabled    bool        `yaml:"enabled" default:"false"`
	NodeRecord node.Config `yaml:"nodeRecord"`
	Auth       AuthConfig  `yaml:"auth"`
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled" default:"false"`
	Secret  string `yaml:"secret"`
}

func (c *AuthConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Secret == "" {
		return errors.New("secret is required")
	}

	return nil
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if err := c.NodeRecord.Validate(); err != nil {
		return err
	}

	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("invalid auth config: %w", err)
	}

	return nil
}
