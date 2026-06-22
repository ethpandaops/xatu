package cannon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/output"
)

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
