package cluster

import (
	"errors"
)

type Config struct {
	Address string `yaml:"address" default:"redis://localhost:6379/0"`
	Prefix  string `yaml:"prefix" default:"xatu"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
