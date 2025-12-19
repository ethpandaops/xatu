package maxmind

import (
	"errors"

	"github.com/ethpandaops/xatu/pkg/geoip/maxmind/database"
	"github.com/ethpandaops/xatu/pkg/geoip/maxmind/geonames"
)

type Config struct {
	Database *database.Config `yaml:"database"`
	GeoNames *geonames.Config `yaml:"geonames"`
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
