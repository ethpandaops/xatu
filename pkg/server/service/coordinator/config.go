package coordinator

import "github.com/ethpandaops/xatu/pkg/server/service/coordinator/persistence"

type Config struct {
	Enabled     bool               `yaml:"enabled" default:"false"`
	Persistence persistence.Config `yaml:"persistence"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if err := c.Persistence.Validate(); err != nil {
		return err
	}

	return nil
}
