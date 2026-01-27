package ethereum

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
)

// BeaconNodeConfig holds configuration for a single beacon node.
type BeaconNodeConfig struct {
	// Name is a human-readable name for this beacon node.
	Name string `yaml:"name"`
	// Address is the HTTP address of the beacon node.
	Address string `yaml:"address"`
	// Headers is a map of headers to send to the beacon node.
	Headers map[string]string `yaml:"headers"`
}

// Validate validates the beacon node configuration.
func (c *BeaconNodeConfig) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	if c.Name == "" {
		return errors.New("name is required")
	}

	return nil
}

// Config holds configuration for the Ethereum beacon node pool.
type Config struct {
	// BeaconNodes is a list of beacon nodes to connect to.
	BeaconNodes []BeaconNodeConfig `yaml:"beaconNodes"`
	// OverrideNetworkName is the name of the network to use.
	// If not set, the network name will be retrieved from the first healthy beacon node.
	OverrideNetworkName string `yaml:"overrideNetworkName" default:""`
	// StartupTimeout is the maximum time to wait for a healthy beacon node on startup.
	StartupTimeout human.Duration `yaml:"startupTimeout" default:"60s"`
	// HealthCheckInterval is the interval between health checks.
	HealthCheckInterval human.Duration `yaml:"healthCheckInterval" default:"3s"`
	// BlockCacheSize is the number of blocks to cache per beacon node.
	BlockCacheSize uint64 `yaml:"blockCacheSize" default:"1000"`
	// BlockCacheTTL is the time to live for blocks in the cache.
	BlockCacheTTL human.Duration `yaml:"blockCacheTtl" default:"1h"`
	// BlockPreloadWorkers is the number of workers to use for preloading blocks.
	BlockPreloadWorkers uint64 `yaml:"blockPreloadWorkers" default:"5"`
	// BlockPreloadQueueSize is the size of the queue for preloading blocks.
	BlockPreloadQueueSize uint64 `yaml:"blockPreloadQueueSize" default:"5000"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if len(c.BeaconNodes) == 0 {
		return errors.New("at least one beacon node is required")
	}

	for i, node := range c.BeaconNodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("invalid beacon node config at index %d: %w", i, err)
		}
	}

	if c.HealthCheckInterval.Duration <= 0 {
		c.HealthCheckInterval.Duration = 3 * time.Second
	}

	if c.StartupTimeout.Duration <= 0 {
		c.StartupTimeout.Duration = 60 * time.Second
	}

	return nil
}
