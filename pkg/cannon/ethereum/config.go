package ethereum

import (
	"errors"

	"github.com/ethpandaops/beacon/pkg/human"
)

type Config struct {
	// The address of the Beacon node to connect to
	BeaconNodeAddress string `yaml:"beaconNodeAddress"`
	// OverrideNetworkName is the name of the network to use for the sentry.
	// If not set, the network name will be retrieved from the beacon node.
	OverrideNetworkName string `yaml:"overrideNetworkName"  default:""`
	// BlockCacheSize is the number of blocks to cache.
	BlockCacheSize uint64 `yaml:"blockCacheSize" default:"1000"`
	// BlockCacheTTL is the time to live for blocks in the cache.
	BlockCacheTTL human.Duration `yaml:"blockCacheTtl" default:"1h"`
	// Headers is a map of headers to send to the beacon node.
	Headers map[string]string `yaml:"headers"`
}

func (c *Config) Validate() error {
	if c.BeaconNodeAddress == "" {
		return errors.New("beaconNodeAddress is required")
	}

	return nil
}
