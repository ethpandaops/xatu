package static

import (
	"errors"
	"time"
)

type Config struct {
	RetryInterval time.Duration `yaml:"retryInterval" default:"60s"`
	NodeRecords   []string      `yaml:"nodeRecords"`
}

func (c *Config) Validate() error {
	if len(c.NodeRecords) == 0 {
		return errors.New("nodeRecords is required")
	}

	return nil
}
