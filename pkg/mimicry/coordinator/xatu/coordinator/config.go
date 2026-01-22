package coordinator

import (
	"errors"

	"github.com/ethpandaops/xatu/pkg/networks"
)

type Config struct {
	Address      string            `yaml:"address"`
	Headers      map[string]string `yaml:"headers"`
	TLS          bool              `yaml:"tls" default:"false"`
	NetworkIds   []uint64          `yaml:"networkIds"`
	ForkIDHashes []string          `yaml:"forkIdHashes"`
	Capabilities []string          `yaml:"capabilities"`
	MaxPeers     uint32            `yaml:"maxPeers" default:"100"`

	// NetworkConfig allows fetching network configuration from a URL.
	// When set, NetworkIds and ForkIDHashes will be computed from the fetched devnet configuration.
	NetworkConfig *networks.DevnetConfig `yaml:"networkConfig"`

	// Enodes contains execution layer enode URLs to use as seed peers.
	// These are populated from networkConfig if available.
	Enodes []string `yaml:"-"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	// NetworkIds can be empty if NetworkConfig is provided
	if len(c.NetworkIds) == 0 && c.NetworkConfig == nil {
		return errors.New("networkIds is required (or provide networkConfig)")
	}

	if c.NetworkConfig != nil {
		return c.NetworkConfig.Validate()
	}

	return nil
}
