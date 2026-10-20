package deriver

import (
	"reflect"
	"strings"
	"testing"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
)

func defaultConfig(t *testing.T) *Config {
	t.Helper()

	cfg := &Config{}
	require.NoError(t, defaults.Set(cfg))

	return cfg
}

// TestConsensusIteratorsCoverEveryDeriver keeps the validation list in step with ConsensusConfig.
func TestConsensusIteratorsCoverEveryDeriver(t *testing.T) {
	cfg := defaultConfig(t)
	iterators := cfg.Consensus.iterators()

	iteratorType := reflect.TypeOf(iterator.BackfillingCheckpointConfig{})
	consensus := reflect.ValueOf(&cfg.Consensus).Elem()

	seen := 0

	for i := range consensus.NumField() {
		field := consensus.Type().Field(i)

		it, ok := field.Type.FieldByName("Iterator")
		if !ok || it.Type != iteratorType {
			continue
		}

		name := strings.Split(field.Tag.Get("yaml"), ",")[0]

		got, ok := iterators[name]
		require.True(t, ok, "consensus.%s is missing from iterators()", name)

		want := consensus.Field(i).FieldByName("Iterator").Addr().Interface()
		assert.Same(t, want, got, "consensus.%s points at another deriver's iterator", name)

		seen++
	}

	assert.Len(t, iterators, seen)
}

func TestValidateEpochsPerBatch(t *testing.T) {
	t.Run("defaults are one epoch per batch", func(t *testing.T) {
		cfg := defaultConfig(t)
		require.NoError(t, cfg.Validate())

		for name, it := range cfg.Consensus.iterators() {
			assert.Equal(t, uint64(1), it.Backfill.EpochsPerBatch, name)
		}
	})

	t.Run("block derivers accept batches", func(t *testing.T) {
		cfg := defaultConfig(t)
		cfg.Consensus.BeaconBlockGraffitiConfig.Iterator.Backfill.EpochsPerBatch = 64
		cfg.Consensus.BeaconBlockConfig.Iterator.Backfill.EpochsPerBatch = iterator.MaxEpochsPerBatch
		require.NoError(t, cfg.Validate())
	})

	t.Run("batch size is bounded", func(t *testing.T) {
		cfg := defaultConfig(t)
		cfg.Consensus.BeaconBlockGraffitiConfig.Iterator.Backfill.EpochsPerBatch = iterator.MaxEpochsPerBatch + 1

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "consensus.beaconBlockGraffiti")
	})

	for _, tt := range []struct {
		name string
		set  func(*Config)
	}{
		{name: "beaconSyncCommittee", set: func(c *Config) { c.Consensus.BeaconSyncCommitteeConfig.Iterator.Backfill.EpochsPerBatch = 2 }},
		{name: "beaconValidators", set: func(c *Config) { c.Consensus.BeaconValidatorsConfig.Iterator.Backfill.EpochsPerBatch = 2 }},
		{name: "blockAccessList", set: func(c *Config) { c.Consensus.BlockAccessListConfig.Iterator.Backfill.EpochsPerBatch = 2 }},
	} {
		t.Run(tt.name+" rejects batches", func(t *testing.T) {
			cfg := defaultConfig(t)
			tt.set(cfg)

			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "consensus."+tt.name)
		})
	}
}
