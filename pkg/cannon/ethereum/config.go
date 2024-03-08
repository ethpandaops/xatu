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
	// BeaconNodeHeaders is a map of headers to send to the beacon node.
	BeaconNodeHeaders map[string]string `yaml:"beaconNodeHeaders"`
	// BlockCacheSize is the number of blocks to cache.
	BlockCacheSize uint64 `yaml:"blockCacheSize" default:"1000"`
	// BlockCacheTTL is the time to live for blocks in the cache.
	BlockCacheTTL human.Duration `yaml:"blockCacheTtl" default:"1h"`
	// BlockPreloadWorkers is the number of workers to use for preloading blocks.
	BlockPreloadWorkers uint64 `yaml:"blockPreloadWorkers" default:"5"`
	// BlockPreloadQueueSize is the size of the queue for preloading blocks.
	BlockPreloadQueueSize uint64 `yaml:"blockPreloadQueueSize" default:"5000"`
	// BlobSidecarsCacheSize is the number of blob sidecars to cache.
	BlobSidecarsCacheSize uint64 `yaml:"blobSidecarsCacheSize" default:"1000"`
	// BlobSidecarsCacheTTL is the time to live for blob sidecars in the cache.
	BlobSidecarsCacheTTL human.Duration `yaml:"blobSidecarsCacheTtl" default:"1h"`
	// BlobSidecarsPreloadWorkers is the number of workers to use for preloading blob sidecars.
	BlobSidecarsPreloadWorkers uint64 `yaml:"blobSidecarsPreloadWorkers" default:"5"`
	// BlobSidecarsPreloadQueueSize is the size of the queue for preloading blob sidecars.
	BlobSidecarsPreloadQueueSize uint64 `yaml:"blobSidecarsPreloadQueueSize" default:"5000"`
}

func (c *Config) Validate() error {
	if c.BeaconNodeAddress == "" {
		return errors.New("beaconNodeAddress is required")
	}

	return nil
}
