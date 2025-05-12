package xatu

import (
	"time"
)

type Config struct {
	Address      string            `yaml:"address"`
	Headers      map[string]string `yaml:"headers"`
	TLS          bool              `yaml:"tls" default:"false"`
	Restart      time.Duration     `yaml:"restart" default:"2m"`
	NetworkIds   []uint64          `yaml:"networkIds"`
	ForkIDHashes []string          `yaml:"forkIdHashes"`
	ForkDigests  []string          `yaml:"forkDigests"`
}

func (c *Config) Validate() error {
	return nil
}
