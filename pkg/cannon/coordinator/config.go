package coordinator

import (
	"errors"
)

type Config struct {
	Address string            `yaml:"address"`
	Headers map[string]string `yaml:"headers"`
	TLS     bool              `yaml:"tls" default:"false"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
