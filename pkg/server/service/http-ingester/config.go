package httpingester

import (
	"fmt"
)

// Config holds configuration for the HTTP ingester server.
// Note: The HTTP ingester reuses services.eventIngester config for auth, outputs, etc.
type Config struct {
	// Enabled indicates whether the HTTP ingester is enabled.
	Enabled bool `yaml:"enabled" default:"false"`
	// Addr is the address to listen on for HTTP requests.
	Addr string `yaml:"addr" default:":8087"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Addr == "" {
		return fmt.Errorf("addr is required when HTTP ingester is enabled")
	}

	return nil
}
