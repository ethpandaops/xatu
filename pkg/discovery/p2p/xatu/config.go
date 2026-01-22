package xatu

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/networks"
)

// ExecutionConfig holds configuration for execution layer node discovery dialing.
type ExecutionConfig struct {
	// RetryAttempts is the maximum number of retry attempts for dialing a peer.
	RetryAttempts uint `yaml:"retryAttempts" default:"5"`
	// RetryDelay is the delay between retry attempts.
	RetryDelay time.Duration `yaml:"retryDelay" default:"5s"`
	// DialTimeout is the timeout for dialing a peer.
	DialTimeout time.Duration `yaml:"dialTimeout" default:"15s"`
}

// ConsensusConfig holds configuration for consensus layer node discovery dialing.
type ConsensusConfig struct {
	// RetryAttempts is the maximum number of retry attempts for dialing a peer.
	RetryAttempts int `yaml:"retryAttempts" default:"1"`
	// RetryDelay is the backoff delay between retry attempts.
	RetryDelay time.Duration `yaml:"retryDelay" default:"2s"`
	// DialTimeout is the timeout for dialing a peer.
	DialTimeout time.Duration `yaml:"dialTimeout" default:"5s"`
	// DialConcurrency is the number of concurrent dial attempts.
	DialConcurrency int `yaml:"dialConcurrency" default:"10"`
	// CooloffDuration is the duration to wait before retrying a failed peer.
	CooloffDuration time.Duration `yaml:"cooloffDuration" default:"10s"`
	// ConnectionTimeout is the timeout for establishing a connection.
	ConnectionTimeout time.Duration `yaml:"connectionTimeout" default:"30s"`
}

type Config struct {
	Address      string            `yaml:"address"`
	Headers      map[string]string `yaml:"headers"`
	TLS          bool              `yaml:"tls" default:"false"`
	DiscV4       bool              `yaml:"discV4" default:"true"`
	DiscV5       bool              `yaml:"discV5" default:"true"`
	Restart      time.Duration     `yaml:"restart" default:"2m"`
	NetworkIds   []uint64          `yaml:"networkIds"`
	ForkIDHashes []string          `yaml:"forkIdHashes"`
	ForkDigests  []string          `yaml:"forkDigests"`
	Execution    ExecutionConfig   `yaml:"execution"`
	Consensus    ConsensusConfig   `yaml:"consensus"`

	// NetworkConfig allows fetching network configuration from a URL.
	// When set, NetworkIds, ForkIDHashes, and ForkDigests will be computed
	// from the fetched devnet configuration.
	NetworkConfig *networks.DevnetConfig `yaml:"networkConfig"`

	// BootNodes contains ENR strings to use as seed boot nodes for discovery.
	// These are populated from networkConfig if available.
	BootNodes []string `yaml:"-"`

	// Enodes contains execution layer enode URLs to use as seed boot nodes.
	// These are populated from networkConfig if available.
	Enodes []string `yaml:"-"`
}

func (c *Config) Validate() error {
	if c.NetworkConfig != nil {
		return c.NetworkConfig.Validate()
	}

	return nil
}
