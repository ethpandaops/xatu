package ethereum

import (
	"errors"

	"github.com/ethpandaops/beacon/pkg/human"
)

// Config is the cannon's Ethereum connection config, grouped by layer.
type Config struct {
	// OverrideNetworkName overrides the network name. If empty, the network is
	// derived from the beacon node.
	OverrideNetworkName string `yaml:"overrideNetworkName" default:""`
	// Beacon configures the consensus-layer (beacon) node. Always required:
	// every consensus deriver reads it, and the execution iterator gates EL
	// progress on CL finality (it never extracts past the CL-finalized block).
	Beacon BeaconConfig `yaml:"beacon"`
	// Execution configures the execution-layer node the EL derivers collect from
	// via cryo. Required only when any execution deriver is enabled.
	Execution ExecutionConfig `yaml:"execution"`
}

// BeaconConfig configures the beacon node connection plus the beacon-block
// cache and preloader.
type BeaconConfig struct {
	// Address is the beacon node to connect to.
	Address string `yaml:"address"`
	// Headers are sent on every beacon node request (e.g. authorization).
	Headers map[string]string `yaml:"headers"`
	// BlockCacheSize is the number of beacon blocks to cache.
	BlockCacheSize uint64 `yaml:"blockCacheSize" default:"1000"`
	// BlockCacheTTL is the time to live for cached beacon blocks.
	BlockCacheTTL human.Duration `yaml:"blockCacheTtl" default:"1h"`
	// BlockPreloadWorkers is the number of workers used to preload blocks.
	BlockPreloadWorkers uint64 `yaml:"blockPreloadWorkers" default:"5"`
	// BlockPreloadQueueSize is the size of the block preload queue.
	BlockPreloadQueueSize uint64 `yaml:"blockPreloadQueueSize" default:"5000"`
}

// ExecutionConfig configures the execution-layer JSON-RPC endpoint that the EL
// derivers feed to cryo (cryo --rpc). Basic-auth credentials may be embedded in
// the address (https://user:pass@host). cryo has no header-injection path, so
// there is intentionally no Headers field here — embed creds in the URL.
type ExecutionConfig struct {
	// Address is the execution-layer JSON-RPC endpoint.
	Address string `yaml:"address"`
}

func (c *Config) Validate() error {
	if c.Beacon.Address == "" {
		return errors.New("ethereum.beacon.address is required")
	}

	return nil
}
