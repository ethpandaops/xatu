package persistence

import (
	"errors"
	"time"
)

type Config struct {
	ConnectionString   string        `yaml:"connectionString"`
	DriverName         DriverName    `yaml:"driverName"`
	MaxIdleConns       int           `yaml:"maxIdleConns" default:"2"`
	MaxOpenConns       int           `yaml:"maxOpenConns" default:"0"`
	MaxQueueSize       int           `yaml:"maxQueueSize" default:"51200"`
	BatchTimeout       time.Duration `yaml:"batchTimeout" default:"5s"`
	ExportTimeout      time.Duration `yaml:"exportTimeout" default:"30s"`
	MaxExportBatchSize int           `yaml:"maxExportBatchSize" default:"512"`
}

func (e *Config) Validate() error {
	if e.ConnectionString == "" {
		return errors.New("connectionString is required")
	}

	return nil
}
