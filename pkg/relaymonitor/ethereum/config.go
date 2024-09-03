package ethereum

import "errors"

type Config struct {
	Network          string `yaml:"network"`
	NetworkConfigURL string `yaml:"networkConfigUrl"`
	GenesisJSONURL   string `yaml:"genesisJsonUrl"`
	SlotsPerEpoch    uint64 `yaml:"slotsPerEpoch"`
}

func (c *Config) Validate() error {
	if c.Network == "" {
		return errors.New("network is required")
	}

	if c.NetworkConfigURL == "" {
		return errors.New("networkConfigUrl is required")
	}

	if c.GenesisJSONURL == "" {
		return errors.New("genesisJsonUrl is required")
	}

	if c.SlotsPerEpoch == 0 {
		return errors.New("slotsPerEpoch is required")
	}

	return nil
}
