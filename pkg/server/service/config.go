package service

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/server/service/coordinator"
	eventingester "github.com/ethpandaops/xatu/pkg/server/service/event-ingester"
)

type Config struct {
	EventIngester eventingester.Config `yaml:"eventIngester"`
	Coordinator   coordinator.Config   `yaml:"coordinator"`
}

func (c *Config) Validate() error {
	if !c.EventIngester.Enabled && !c.Coordinator.Enabled {
		return fmt.Errorf("no services configured")
	}

	if err := c.EventIngester.Validate(); err != nil {
		return err
	}

	if err := c.Coordinator.Validate(); err != nil {
		return err
	}

	return nil
}
