package http

import "errors"

type Config struct {
	Address              string `yaml:"address"`
	BatchSize            int    `yaml:"batch_size" default:"1000"`
	BatchIntervalSeconds int    `yaml:"batch_interval_seconds" default:"5"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
