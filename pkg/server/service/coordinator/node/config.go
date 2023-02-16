package node

import (
	"time"
)

type Config struct {
	MaxQueueSize       int           `yaml:"maxQueueSize" default:"51200"`
	BatchTimeout       time.Duration `yaml:"batchTimeout" default:"5s"`
	ExportTimeout      time.Duration `yaml:"exportTimeout" default:"30s"`
	MaxExportBatchSize int           `yaml:"maxExportBatchSize" default:"512"`
}

func (e *Config) Validate() error {
	return nil
}
