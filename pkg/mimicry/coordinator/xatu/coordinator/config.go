package coordinator

import (
	"errors"
)

type Config struct {
	Address      string            `yaml:"address"`
	Headers      map[string]string `yaml:"headers"`
	TLS          bool              `yaml:"tls" default:"false"`
	NetworkIds   []uint64          `yaml:"networkIds"`
	ForkIDHashes []string          `yaml:"forkIdHashes"`
	Capabilities []string          `yaml:"capabilities"`
	MaxPeers     uint32            `yaml:"maxPeers" default:"100"`
}

func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	if len(c.NetworkIds) == 0 {
		return errors.New("networkIds is required")
	}

	return nil
}
