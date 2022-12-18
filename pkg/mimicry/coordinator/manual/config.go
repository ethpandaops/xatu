package manual

import (
	"errors"
	"time"
)

type Config struct {
	RetryInterval time.Duration `yaml:"retry_interval" default:"60s"`
	NodeRecords   []string      `yaml:"node_records"`
}

func (c *Config) Validate() error {
	if len(c.NodeRecords) == 0 {
		return errors.New("node_records is required")
	}

	return nil
}
