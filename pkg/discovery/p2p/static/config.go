package static

import (
	"errors"
	"time"
)

// ExecutionConfig holds configuration for execution layer node discovery dialing.
type ExecutionConfig struct {
	// RetryAttempts is the maximum number of retry attempts for dialing a peer.
	// Execution clients throttle repeat connections from the same IP (30s for
	// geth/erigon/reth, 5m per /24 for nethermind), so rapid in-process retries
	// are rejected before the handshake; rely on rediscovery instead.
	RetryAttempts uint `yaml:"retryAttempts" default:"1"`
	// RetryDelay is the delay between retry attempts, when retries are enabled.
	// Must exceed the remote client's same-IP reconnect throttle to be useful.
	RetryDelay time.Duration `yaml:"retryDelay" default:"60s"`
	// DialTimeout is the timeout for dialing a peer.
	DialTimeout time.Duration `yaml:"dialTimeout" default:"15s"`
	// BootstrapRPCURL is an optional execution JSON-RPC endpoint used to build
	// our own Status message and serve lightweight block/header/body/receipt
	// requests during the eth handshake. When set, discovery becomes a more
	// useful peer and recovers connections that would otherwise drop after
	// Status exchange.
	BootstrapRPCURL string `yaml:"bootstrapRpcUrl" default:""`
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
	BootNodes []string        `yaml:"bootNodes"`
	DiscV4    bool            `yaml:"discV4" default:"true"`
	DiscV5    bool            `yaml:"discV5" default:"true"`
	Restart   time.Duration   `yaml:"restart" default:"2m"`
	Execution ExecutionConfig `yaml:"execution"`
	Consensus ConsensusConfig `yaml:"consensus"`
}

func (c *Config) Validate() error {
	if len(c.BootNodes) == 0 {
		return errors.New("bootNodes is required")
	}

	return nil
}
