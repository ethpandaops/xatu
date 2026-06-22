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
// structs (it exercises the cryo / ethereum.executionNodeAddress /
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
			Ethereum:    ethereum.Config{BeaconNodeAddress: "http://localhost:5052"},
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
