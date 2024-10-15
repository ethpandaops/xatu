package iterator

import "github.com/attestantio/go-eth2-client/spec/phase0"

type BackfillingCheckpointConfig struct {
	Backfill struct {
		Enabled bool         `yaml:"enabled" default:"false"`
		ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
	} `yaml:"backfill"`
}
