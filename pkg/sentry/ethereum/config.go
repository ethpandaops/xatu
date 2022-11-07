package ethereum

import "errors"

type Config struct {
	// The address of the Beacon node to connect to
	BeaconNodeAddress string `yaml:"beacon_node_address"`
	// Network name of the Ethereum network we are expecting to connect to
	Network string `yaml:"network"`
	// ConsensusClient is the name of the consensus client we are connecting to
	ConsensusClient string `yaml:"consensus_client"`
	// ExecutionClient is the name of the execution client we are connecting to
	ExecutionClient string `yaml:"execution_client"`
}

func (c *Config) Validate() error {
	if c.BeaconNodeAddress == "" {
		return errors.New("beacon_node_address is required")
	}

	if c.Network == "" {
		return errors.New("network is required")
	}

	if c.ConsensusClient == "" {
		return errors.New("consensus_client is required")
	}

	if c.ExecutionClient == "" {
		return errors.New("execution_client is required")
	}

	return nil
}
