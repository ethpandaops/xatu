package service

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/server/service/contributoor"
	"github.com/ethpandaops/xatu/pkg/server/service/coordinator"
	eventingester "github.com/ethpandaops/xatu/pkg/server/service/event-ingester"
)

type Config struct {
	EventIngester eventingester.Config `yaml:"eventIngester"`
	Coordinator   coordinator.Config   `yaml:"coordinator"`
	Contributoor  contributoor.Config  `yaml:"contributoor"`
}

func (c *Config) Validate() error {
	if !c.EventIngester.Enabled && !c.Coordinator.Enabled && !c.Contributoor.Enabled {
		return fmt.Errorf("no services configured")
	}

	if err := c.EventIngester.Validate(); err != nil {
		return err
	}

	if err := c.Coordinator.Validate(); err != nil {
		return err
	}

	if err := c.Contributoor.Validate(); err != nil {
		return err
	}

	return nil
}
