package p2p

import (
	"errors"
	"time"
)

type Config struct {
	BootNodes []string      `yaml:"boot_nodes"`
	DiscV4    bool          `yaml:"discV4" default:"true"`
	DiscV5    bool          `yaml:"discV5" default:"true"`
	Restart   time.Duration `yaml:"restart" default:"2m"`
}

func (c *Config) Validate() error {
	if len(c.BootNodes) == 0 {
		return errors.New("boot_nodes is required")
	}

	return nil
}
