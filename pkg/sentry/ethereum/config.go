package ethereum

import "errors"

type Config struct {
	// The address of the Beacon node to connect to
	BeaconNodeAddress string `yaml:"beaconNodeAddress"`
}

func (c *Config) Validate() error {
	if c.BeaconNodeAddress == "" {
		return errors.New("beaconNodeAddress is required")
	}

	return nil
}
