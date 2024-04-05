package xatu

import (
	"errors"
	"time"
)

type Config struct {
	Address            string            `yaml:"address"`
	Headers            map[string]string `yaml:"headers"`
	TLS                bool              `yaml:"tls" default:"false"`
	MaxQueueSize       int               `yaml:"maxQueueSize" default:"51200"`
	BatchTimeout       time.Duration     `yaml:"batchTimeout" default:"5s"`
	ExportTimeout      time.Duration     `yaml:"exportTimeout" default:"30s"`
	MaxExportBatchSize int               `yaml:"maxExportBatchSize" default:"512"`
	Workers            int               `yaml:"workers" default:"1"`
	Retry              RetryConfig       `yaml:"retry"`
}

type RetryConfig struct {
	Interval    time.Duration `yaml:"interval" default:"5s"`
	MaxAttempts int           `yaml:"maxAttempts" default:"5"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
