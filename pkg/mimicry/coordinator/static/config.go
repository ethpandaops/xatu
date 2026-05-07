package static

import (
	"errors"
	"time"
)

type Config struct {
	RetryInterval      time.Duration `yaml:"retryInterval" default:"60s"`
	MaxConcurrentPeers int           `yaml:"maxConcurrentPeers" default:"0"`
	NodeRecords        []string      `yaml:"nodeRecords"`
}

func (c *Config) Validate() error {
	if len(c.NodeRecords) == 0 {
		return errors.New("nodeRecords is required")
	}

	if c.MaxConcurrentPeers < 0 {
		return errors.New("maxConcurrentPeers cannot be negative")
	}

	return nil
}
