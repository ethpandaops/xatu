package httpingester

import (
	"fmt"

	eventingester "github.com/ethpandaops/xatu/pkg/server/service/event-ingester"
)

// Config holds configuration for the HTTP ingester server.
type Config struct {
	// Enabled indicates whether the HTTP ingester is enabled.
	Enabled bool `yaml:"enabled" default:"false"`
	// Addr is the address to listen on for HTTP requests.
	Addr string `yaml:"addr" default:":8081"`
	// EventIngester holds the event ingester configuration (auth, outputs, etc.)
	// This is shared with the gRPC event ingester.
	EventIngester *eventingester.Config `yaml:"eventIngester"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Addr == "" {
		return fmt.Errorf("addr is required when HTTP ingester is enabled")
	}

	if c.EventIngester == nil {
		return fmt.Errorf("eventIngester config is required when HTTP ingester is enabled")
	}

	return nil
}
