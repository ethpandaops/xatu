package maxmind

import (
	"errors"

	"github.com/ethpandaops/xatu/pkg/server/geoip/maxmind/database"
)

type Config struct {
	Database *database.Config `yaml:"database"`
}

func (c *Config) Validate() error {
	if c.Database == nil {
		return errors.New("database is required")
	}

	if err := c.Database.Validate(); err != nil {
		return err
	}

	return nil
}
