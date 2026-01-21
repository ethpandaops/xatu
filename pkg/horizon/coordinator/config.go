package coordinator

import (
	"errors"
)

// Config holds the configuration for the coordinator client.
type Config struct {
	// Address is the gRPC address of the coordinator server.
	Address string `yaml:"address"`
	// Headers are optional headers to send with each request.
	Headers map[string]string `yaml:"headers"`
	// TLS enables TLS for the gRPC connection.
	TLS bool `yaml:"tls" default:"false"`
}

// Validate validates the coordinator configuration.
func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
