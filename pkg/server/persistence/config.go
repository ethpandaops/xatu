package persistence

import (
	"errors"
)

type Config struct {
	Enabled          bool       `yaml:"enabled" default:"false"`
	ConnectionString string     `yaml:"connectionString"`
	DriverName       DriverName `yaml:"driverName"`
	MaxIdleConns     int        `yaml:"maxIdleConns" default:"2"`
	MaxOpenConns     int        `yaml:"maxOpenConns" default:"0"`
}

func (e *Config) Validate() error {
	if !e.Enabled {
		return nil
	}

	if e.ConnectionString == "" {
		return errors.New("connectionString is required")
	}

	return nil
}
