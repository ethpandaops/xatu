package ethereum

import "errors"

type Config struct {
	Network       string         `yaml:"network"`
	CustomNetwork *CustomNetwork `yaml:"customNetwork" default:""`
}

func (c *Config) Validate() error {
	if c.Network == "" {
		return errors.New("network is required")
	}

	return nil
}

type CustomNetwork struct {
	ConfigURL               string `yaml:"configUrl"`
	BootnodeENRURL          string `yaml:"bootnodeEnrUrl"`
	DepositContractBlockURL string `yaml:"depositContractBlockUrl"`
}

func (c *CustomNetwork) Validate() error {
	if c.ConfigURL == "" {
		return errors.New("configUrl is required")
	}

	if c.BootnodeENRURL == "" {
		return errors.New("bootnodeUrl is required")
	}

	if c.DepositContractBlockURL == "" {
		return errors.New("depositContractBlockUrl is required")
	}

	return nil
}
