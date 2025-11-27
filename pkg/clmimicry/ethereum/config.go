package ethereum

import (
	"errors"

	"github.com/OffchainLabs/prysm/v7/config/params"
)

type Config struct {
	Network string `yaml:"network"`

	// Devnet specific attributes (requires 'network=devnet').
	Devnet DevnetOptions `yaml:"devnet"`
}

type DevnetOptions struct {
	// ConfigURL is the .yaml URL to fetch the beacon chain cfg.
	ConfigURL string `yaml:"configUrl"`
	// BootnodesURL is the .yaml URL from which to fetch the bootnode ENRs.
	BootnodesURL string `yaml:"bootnodesUrl"`
	// DepositContractBlockURL is the .txt URL from which to fetch the deposit contract block.
	DepositContractBlockURL string `yaml:"depositContractBlockUrl"`
	// GenesisSSZURL is the .ssz URL from which to fetch the genesis data.
	GenesisSSZURL string `yaml:"genesisSszUrl"`
}

func (c *Config) Validate() error {
	if c.Network == "" {
		return errors.New("ethereum.network is required")
	}

	if c.Network != params.DevnetName {
		return nil
	}

	if c.Devnet.ConfigURL == "" {
		return errors.New("ethereum.network.devnet.configUrl is required")
	}

	if c.Devnet.BootnodesURL == "" {
		return errors.New("ethereum.network.devnet.bootnodesUrl is required")
	}

	if c.Devnet.DepositContractBlockURL == "" {
		return errors.New("ethereum.network.devnet.depositContractBlockUrl is required")
	}

	if c.Devnet.GenesisSSZURL == "" {
		return errors.New("ethereum.network.devnet.genesisSszUrl is required")
	}

	return nil
}
