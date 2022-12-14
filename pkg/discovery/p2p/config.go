package p2p

import "errors"

type Config struct {
	BootNodes []string `yaml:"boot_nodes"`
}

func (c *Config) Validate() error {
	if len(c.BootNodes) == 0 {
		return errors.New("boot_nodes is required")
	}

	return nil
}
