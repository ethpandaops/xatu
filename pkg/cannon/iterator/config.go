package iterator

import (
	"fmt"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// MaxEpochsPerBatch bounds iterator.backfill.epochsPerBatch: a batch is held in memory until it is written.
const MaxEpochsPerBatch = 256

type BackfillingCheckpointConfig struct {
	Backfill struct {
		Enabled bool         `yaml:"enabled" default:"false"`
		ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
		// EpochsPerBatch is how many epochs one backfill step derives and writes before the marker advances.
		EpochsPerBatch uint64 `yaml:"epochsPerBatch" default:"1"`
	} `yaml:"backfill"`
}

// Validate checks the iterator config.
func (c *BackfillingCheckpointConfig) Validate() error {
	if c.Backfill.EpochsPerBatch > MaxEpochsPerBatch {
		return fmt.Errorf("iterator.backfill.epochsPerBatch must be at most %d, got %d", MaxEpochsPerBatch, c.Backfill.EpochsPerBatch)
	}

	return nil
}

// epochsPerBatch is the configured backfill batch size, with zero meaning one epoch.
func (c *BackfillingCheckpointConfig) epochsPerBatch() uint64 {
	return max(1, c.Backfill.EpochsPerBatch)
}
