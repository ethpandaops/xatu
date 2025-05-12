package static

import (
	"errors"
	"time"
)

type Config struct {
	BootNodes []string      `yaml:"bootNodes"`
	DiscV5    bool          `yaml:"discV5" default:"true"`
	Restart   time.Duration `yaml:"restart" default:"2m"`
}

func (c *Config) Validate() error {
	if len(c.BootNodes) == 0 {
		return errors.New("bootNodes is required")
	}

	return nil
}
