package ethstats

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/ethstats/auth"
)

type Config struct {
	Enabled      bool   `yaml:"enabled" default:"true"`
	Addr         string `yaml:"addr" default:":8081"`
	MetricsAddr  string `yaml:"metricsAddr" default:":9090"`
	LoggingLevel string `yaml:"logging" default:"info"`

	// WebSocket settings
	MaxMessageSize int64         `yaml:"maxMessageSize" default:"15728640"` // 15MB
	ReadTimeout    time.Duration `yaml:"readTimeout" default:"60s"`
	WriteTimeout   time.Duration `yaml:"writeTimeout" default:"10s"`
	PingInterval   time.Duration `yaml:"pingInterval" default:"30s"`

	// Authentication
	Auth auth.Config `yaml:"auth"`

	// Rate limiting
	RateLimit RateLimitConfig `yaml:"rateLimit"`

	// Labels for metrics
	Labels map[string]string `yaml:"labels"`
}

type RateLimitConfig struct {
	Enabled            bool          `yaml:"enabled" default:"true"`
	ConnectionsPerIP   int           `yaml:"connectionsPerIp" default:"10"`
	WindowDuration     time.Duration `yaml:"windowDuration" default:"1m"`
	FailuresBeforeWarn int           `yaml:"failuresBeforeWarn" default:"5"`
}

func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("addr is required")
	}

	if c.MaxMessageSize <= 0 {
		return fmt.Errorf("maxMessageSize must be positive")
	}

	if c.ReadTimeout <= 0 {
		return fmt.Errorf("readTimeout must be positive")
	}

	if c.WriteTimeout <= 0 {
		return fmt.Errorf("writeTimeout must be positive")
	}

	if c.PingInterval <= 0 {
		return fmt.Errorf("pingInterval must be positive")
	}

	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("auth config validation failed: %w", err)
	}

	if err := c.RateLimit.Validate(); err != nil {
		return fmt.Errorf("rate limit config validation failed: %w", err)
	}

	return nil
}

func (c *RateLimitConfig) Validate() error {
	if c.Enabled {
		if c.ConnectionsPerIP <= 0 {
			return fmt.Errorf("connectionsPerIP must be positive when rate limiting is enabled")
		}

		if c.WindowDuration <= 0 {
			return fmt.Errorf("windowDuration must be positive when rate limiting is enabled")
		}

		if c.FailuresBeforeWarn < 0 {
			return fmt.Errorf("failuresBeforeWarn must be non-negative")
		}
	}

	return nil
}

func NewDefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		Addr:           ":8081",
		MetricsAddr:    ":9090",
		LoggingLevel:   "info",
		MaxMessageSize: 15728640, // 15MB
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   10 * time.Second,
		PingInterval:   30 * time.Second,
		Auth: auth.Config{
			Enabled: true,
		},
		RateLimit: RateLimitConfig{
			Enabled:            true,
			ConnectionsPerIP:   10,
			WindowDuration:     time.Minute,
			FailuresBeforeWarn: 5,
		},
		Labels: make(map[string]string),
	}
}
