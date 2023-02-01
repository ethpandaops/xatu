package coordinator

import (
	"errors"
)

type Config struct {
	Address      string            `yaml:"address"`
	Headers      map[string]string `yaml:"headers"`
	TLS          bool              `yaml:"tls" default:"false"`
	NetworkIDs   []uint64          `yaml:"network_ids"`
	ForkIDHashes []string          `yaml:"fork_id_hashes"`
	MaxPeers     uint32            `yaml:"max_peers" default:"100"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	if len(c.NetworkIDs) == 0 {
		return errors.New("network_ids is required")
	}

	return nil
}
