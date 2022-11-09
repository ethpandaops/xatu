package http

import "errors"

type Config struct {
	Address              string            `yaml:"address"`
	Headers              map[string]string `yaml:"headers"`
	TimeoutSeconds       int               `yaml:"timeout_seconds"`
	BatchSize            int               `yaml:"batch_size" default:"1000"`
	BatchIntervalSeconds int               `yaml:"batch_interval_seconds" default:"5"`
	MaxItems             int               `yaml:"max_items" default:"100000"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
