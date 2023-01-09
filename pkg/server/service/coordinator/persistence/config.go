package persistence

import (
	"errors"
	"time"
)

type Config struct {
	ConnectionString   string        `yaml:"connection_string"`
	DriverName         DriverName    `yaml:"driver_name"`
	MaxQueueSize       int           `yaml:"max_queue_size" default:"51200"`
	BatchTimeout       time.Duration `yaml:"batch_timeout" default:"5s"`
	ExportTimeout      time.Duration `yaml:"export_timeout" default:"30s"`
	MaxExportBatchSize int           `yaml:"max_export_batch_size" default:"512"`
}

func (e *Config) Validate() error {
	if e.ConnectionString == "" {
		return errors.New("connection_string is required")
	}

	return nil
}
