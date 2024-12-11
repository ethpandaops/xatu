package ethereum

import "errors"

type Config struct {
	Network           string            `yaml:"network"`
	BeaconNodeURL     string            `yaml:"beaconNodeUrl"`
	BeaconNodeHeaders map[string]string `yaml:"beaconNodeHeaders"`
}

func (c *Config) Validate() error {
	if c.Network == "" {
		return errors.New("network is required")
	}

	if c.BeaconNodeURL == "" {
		return errors.New("beaconNodeUrl is required")
	}

	return nil
}
