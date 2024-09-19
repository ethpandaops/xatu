package ethereum

import "errors"

type Config struct {
	// The address of the Beacon node to connect to
	BeaconNodeAddress string `yaml:"beaconNodeAddress"`
	// Network is the network to use for the beacon node.
	Network string `yaml:"network"`
	// BeaconNodeHeaders is a map of headers to send to the beacon node.
	BeaconNodeHeaders map[string]string `yaml:"beaconNodeHeaders"`
}

func (c *Config) Validate() error {
	if c.BeaconNodeAddress == "" {
		return errors.New("beaconNodeAddress is required")
	}

	return nil
}
