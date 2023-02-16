package coordinator

import "github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"

type Config struct {
	Enabled    bool        `yaml:"enabled" default:"false"`
	NodeRecord node.Config `yaml:"nodeRecord"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if err := c.NodeRecord.Validate(); err != nil {
		return err
	}

	return nil
}
