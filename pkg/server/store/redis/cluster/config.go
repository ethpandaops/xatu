package cluster

import (
	"errors"
	"time"
)

type Config struct {
	Address string        `yaml:"addresses" default:"redis://localhost:6379/0"`
	Timeout time.Duration `yaml:"timeout" default:"15s"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
