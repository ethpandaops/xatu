package ethereum

import "errors"

type Config struct {
	Network string `yaml:"network"`
}

func (c *Config) Validate() error {
	if c.Network == "" {
		return errors.New("network is required")
	}

	return nil
}
