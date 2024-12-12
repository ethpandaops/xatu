package registrations

import (
	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/pkg/errors"
)

type Config struct {
	// Enabled is the flag to enable the validator registration monitor
	Enabled *bool `yaml:"enabled" default:"false"`
	// NumShards is the number of shards to split the validator set into
	NumShards int `yaml:"numShards" default:"1"`
	// Shard is the shard to use for the validator set
	Shard int `yaml:"shard" default:"0"`
	// ActiveValidatorsFetchInterval is the interval at which to fetch active validators
	ActiveValidatorsFetchInterval human.Duration `yaml:"activeValidatorsFetchInterval" default:"24h"`
	// TargetSweepDuration is the duration at which we want to sweep all validators in this shard of
	// the validator set.
	// If this is set too low for the size of the validator set, the relays may ban you.
	// Defaults to 14 days. This means that we will sweep all validators in this shard
	// over the course of 14 days.
	TargetSweepDuration human.Duration `yaml:"targetSweepDuration" default:"336h"`
	// Workers is the number of workers to use for fetching active validators per relay
	Workers int `yaml:"workers" default:"10"`
}

func (c *Config) Validate() error {
	if c.NumShards < 1 {
		return errors.New("numShards must be greater than 0")
	}

	if c.Shard < 0 || c.Shard >= c.NumShards {
		return errors.New("shard must be between 0 and numShards")
	}

	return nil
}
