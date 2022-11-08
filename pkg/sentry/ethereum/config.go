package ethereum

import "errors"

type Config struct {
	// The address of the Beacon node to connect to
	BeaconNodeAddress string `yaml:"beacon_node_address"`
	// Network name of the Ethereum network we are expecting to connect to
	Network string `yaml:"network"`
}

func (c *Config) Validate() error {
	if c.BeaconNodeAddress == "" {
		return errors.New("beacon_node_address is required")
	}

	if c.Network == "" {
		return errors.New("network is required")
	}

	return nil
}
