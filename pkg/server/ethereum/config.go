package ethereum

import (
	"github.com/pkg/errors"
)

type Config struct {
	Network NetworkConfig `yaml:"network"`
}

type NetworkConfig struct {
	Name string     `yaml:"name"`
	ID   uint64     `yaml:"id"`
	Spec SpecConfig `yaml:"spec"`
}

type SpecConfig struct {
	SecondsPerSlot uint64 `yaml:"secondsPerSlot"`
	SlotsPerEpoch  uint64 `yaml:"slotsPerEpoch"`
	GenesisTime    uint64 `yaml:"genesisTime"`
}

func (c *Config) Validate() error {
	if err := c.Network.Validate(); err != nil {
		return errors.Wrap(err, "invalid network config")
	}

	return nil
}

func (c *NetworkConfig) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if c.ID == 0 {
		return errors.New("id is required")
	}

	if err := c.Spec.Validate(); err != nil {
		return errors.Wrap(err, "invalid spec config")
	}

	return nil
}

func (c *SpecConfig) Validate() error {
	if c.SecondsPerSlot == 0 {
		return errors.New("seconds_per_slot is required")
	}

	if c.SlotsPerEpoch == 0 {
		return errors.New("slots_per_epoch is required")
	}

	if c.GenesisTime == 0 {
		return errors.New("genesis_time is required")
	}

	return nil
}
