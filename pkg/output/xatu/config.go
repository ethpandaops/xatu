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
	Connections        int               `yaml:"connections" default:"1"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
