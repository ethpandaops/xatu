package cannon

import (
	"os"
	"testing"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"

	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/output"
)

// TestExampleConfigParsesAndValidates loads example_cannon.yaml exactly as the
// cannon command does, so the shipped example can't drift from the config
// structs (it exercises the cryo / ethereum.beacon / ethereum.execution /
// derivers.consensus / derivers.execution layout end to end).
func TestExampleConfigParsesAndValidates(t *testing.T) {
	config := &Config{}
	require.NoError(t, defaults.Set(config))

	data, err := os.ReadFile("../../example_cannon.yaml")
	require.NoError(t, err)

	type plain Config

	require.NoError(t, yaml.Unmarshal(data, (*plain)(config)))

	require.NoError(t, config.Validate())
}

// TestConfig_Validate_RejectsXatuServerOutput asserts cannon no longer accepts
// the xatu-server output (for any data) and steers users to clickhouse.
func TestConfig_Validate_RejectsXatuServerOutput(t *testing.T) {
	mk := func(sink output.SinkType) *Config {
		return &Config{
			Name:        "test",
			Ethereum:    ethereum.Config{Beacon: ethereum.BeaconConfig{Address: "http://localhost:5052"}},
			Coordinator: coordinator.Config{Address: "localhost:8080"},
			Outputs:     []output.Config{{Name: "out", SinkType: sink}},
		}
	}

	t.Run("xatu-server output is rejected", func(t *testing.T) {
		err := mk(output.SinkTypeXatu).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer supported")
	})

	t.Run("clickhouse output is accepted", func(t *testing.T) {
		require.NoError(t, mk(output.SinkTypeClickhouse).Validate())
	})

	t.Run("kafka output is not blocked by the xatu guard", func(t *testing.T) {
		require.NoError(t, mk(output.SinkTypeKafka).Validate())
	})
}

// TestConfig_Validate_BlockAccessListBatchCap asserts an unusable
// maxRowsPerBatch is rejected at startup rather than silently ignored.
func TestConfig_Validate_BlockAccessListBatchCap(t *testing.T) {
	mk := func(maxRows int) *Config {
		cfg := &Config{
			Name:        "test",
			Ethereum:    ethereum.Config{Beacon: ethereum.BeaconConfig{Address: "http://localhost:5052"}},
			Coordinator: coordinator.Config{Address: "localhost:8080"},
			Outputs:     []output.Config{{Name: "out", SinkType: output.SinkTypeClickhouse}},
		}

		cfg.Derivers.Consensus.BlockAccessListConfig.Enabled = true
		cfg.Derivers.Consensus.BlockAccessListConfig.MaxRowsPerBatch = maxRows

		return cfg
	}

	tests := []struct {
		name    string
		maxRows int
		wantErr bool
	}{
		{name: "positive", maxRows: 50000},
		{name: "zero", maxRows: 0, wantErr: true},
		{name: "negative", maxRows: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mk(tt.maxRows).Validate()
			if !tt.wantErr {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), "consensus.blockAccessList")
			assert.Contains(t, err.Error(), "maxRowsPerBatch")
		})
	}
}

// TestConfig_GloasDerivers asserts the Gloas derivers are on by default and
// can be toggled independently through the shipped config keys.
func TestConfig_GloasDerivers(t *testing.T) {
	t.Run("enabled by default", func(t *testing.T) {
		config := &Config{}
		require.NoError(t, defaults.Set(config))

		consensus := config.Derivers.Consensus
		assert.True(t, consensus.ExecutionRequestBuilderDepositConfig.Enabled)
		assert.True(t, consensus.ExecutionRequestBuilderExitConfig.Enabled)
		assert.True(t, consensus.BlockAccessListSummaryConfig.Enabled)
	})

	t.Run("summary runs without the raw block access list deriver", func(t *testing.T) {
		config := &Config{}
		require.NoError(t, defaults.Set(config))

		type plain Config

		require.NoError(t, yaml.Unmarshal([]byte(`
derivers:
  consensus:
    blockAccessList: { enabled: false }
    blockAccessListSummary: { enabled: true }
    executionRequestBuilderDeposit: { enabled: false }
    executionRequestBuilderExit: { enabled: false }
`), (*plain)(config)))

		consensus := config.Derivers.Consensus
		assert.False(t, consensus.BlockAccessListConfig.Enabled)
		assert.True(t, consensus.BlockAccessListSummaryConfig.Enabled)
		assert.False(t, consensus.ExecutionRequestBuilderDepositConfig.Enabled)
		assert.False(t, consensus.ExecutionRequestBuilderExitConfig.Enabled)
	})
}
