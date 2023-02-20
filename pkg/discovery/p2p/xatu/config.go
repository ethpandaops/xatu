package xatu

import (
	"time"
)

type Config struct {
	Address      string            `yaml:"address"`
	Headers      map[string]string `yaml:"headers"`
	TLS          bool              `yaml:"tls" default:"false"`
	DiscV4       bool              `yaml:"discV4" default:"true"`
	DiscV5       bool              `yaml:"discV5" default:"true"`
	Restart      time.Duration     `yaml:"restart" default:"2m"`
	NetworkIDs   []uint64          `yaml:"networkIDs"`
	ForkIDHashes []string          `yaml:"forkIdHashes"`
}

func (c *Config) Validate() error {
	return nil
}
