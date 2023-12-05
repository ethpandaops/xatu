package ethereum

import "errors"

type Config struct {
	// The address of the Beacon node to connect to
	BeaconNodeAddress string `yaml:"beaconNodeAddress"`
	// OverrideNetworkName is the name of the network to use.
	// If not set, the network name will be retrieved from the beacon node.
	OverrideNetworkName string `yaml:"overrideNetworkName"  default:""`
	// BeaconNodeHeaders is a map of headers to send to the beacon node.
	BeaconNodeHeaders map[string]string `yaml:"beaconNodeHeaders"`
}

func (c *Config) Validate() error {
	if c.BeaconNodeAddress == "" {
		return errors.New("beaconNodeAddress is required")
	}

	return nil
}
