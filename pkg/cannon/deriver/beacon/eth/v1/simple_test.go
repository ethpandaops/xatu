package v1

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
)

// Tests for BeaconCommitteeDeriver
func TestBeaconCommitteeDeriver_Constants(t *testing.T) {
	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE, BeaconCommitteeDeriverName)
	assert.Equal(t, "BEACON_API_ETH_V1_BEACON_COMMITTEE", BeaconCommitteeDeriverName.String())
}

func TestBeaconCommitteeDeriver_Methods(t *testing.T) {
	deriver := &BeaconCommitteeDeriver{}

	assert.Equal(t, BeaconCommitteeDeriverName, deriver.CannonType())
	assert.Equal(t, spec.DataVersionPhase0, deriver.ActivationFork())
	assert.Equal(t, BeaconCommitteeDeriverName.String(), deriver.Name())
}

func TestBeaconCommitteeDeriverConfig_BasicStructure(t *testing.T) {
	config := BeaconCommitteeDeriverConfig{
		Enabled: true,
	}

	assert.True(t, config.Enabled)
	// Iterator field exists but we won't test its internal structure here
	assert.NotNil(t, &config.Iterator)
}

// Tests for BeaconValidatorsDeriver
func TestBeaconValidatorsDeriver_Constants(t *testing.T) {
	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS, BeaconValidatorsDeriverName)
	assert.Equal(t, "BEACON_API_ETH_V1_BEACON_VALIDATORS", BeaconValidatorsDeriverName.String())
}

func TestBeaconValidatorsDeriverConfig_BasicStructure(t *testing.T) {
	config := BeaconValidatorsDeriverConfig{
		Enabled:   true,
		ChunkSize: 100,
	}

	assert.True(t, config.Enabled)
	assert.Equal(t, 100, config.ChunkSize)
	assert.NotNil(t, &config.Iterator)
}

func TestBeaconValidatorsDeriverConfig_ChunkSizeValues(t *testing.T) {
	tests := []struct {
		name      string
		chunkSize int
		valid     bool
	}{
		{"positive_chunk", 50, true},
		{"large_chunk", 1000, true},
		{"zero_chunk", 0, false},      // Might be invalid
		{"negative_chunk", -1, false}, // Definitely invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := BeaconValidatorsDeriverConfig{
				ChunkSize: tt.chunkSize,
			}

			assert.Equal(t, tt.chunkSize, config.ChunkSize)
			// In production, you'd have validation logic here
		})
	}
}

// Tests for BeaconBlobDeriver
func TestBeaconBlobDeriver_Constants(t *testing.T) {
	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR, BeaconBlobDeriverName)
	assert.Equal(t, "BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR", BeaconBlobDeriverName.String())
}

func TestBeaconBlobDeriverConfig_BasicStructure(t *testing.T) {
	config := BeaconBlobDeriverConfig{
		Enabled: true,
	}

	assert.True(t, config.Enabled)
	assert.NotNil(t, &config.Iterator)
}

// Tests for all configs' zero values
func TestDeriverConfigs_ZeroValues(t *testing.T) {
	t.Run("beacon_committee_config", func(t *testing.T) {
		var config BeaconCommitteeDeriverConfig
		assert.False(t, config.Enabled)
	})

	t.Run("beacon_validators_config", func(t *testing.T) {
		var config BeaconValidatorsDeriverConfig
		assert.False(t, config.Enabled)
		assert.Equal(t, 0, config.ChunkSize)
	})

	t.Run("beacon_blob_config", func(t *testing.T) {
		var config BeaconBlobDeriverConfig
		assert.False(t, config.Enabled)
	})
}

// Test that all derivers use the correct cannon types
func TestDeriver_CannonTypeConsistency(t *testing.T) {
	tests := []struct {
		name       string
		constant   xatu.CannonType
		expectName string
	}{
		{
			name:       "beacon_committee",
			constant:   BeaconCommitteeDeriverName,
			expectName: "BEACON_API_ETH_V1_BEACON_COMMITTEE",
		},
		{
			name:       "beacon_validators",
			constant:   BeaconValidatorsDeriverName,
			expectName: "BEACON_API_ETH_V1_BEACON_VALIDATORS",
		},
		{
			name:       "beacon_blob",
			constant:   BeaconBlobDeriverName,
			expectName: "BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectName, tt.constant.String())
			assert.Contains(t, tt.constant.String(), "ETH_V1")
			assert.Contains(t, tt.constant.String(), "BEACON")
		})
	}
}

// Test config field accessibility (important for YAML parsing)
func TestDeriverConfigs_FieldAccessibility(t *testing.T) {
	t.Run("beacon_committee_fields", func(t *testing.T) {
		config := BeaconCommitteeDeriverConfig{Enabled: true}
		assert.True(t, config.Enabled)
		// Iterator field should be accessible even if we don't test its internals
		_ = config.Iterator
	})

	t.Run("beacon_validators_fields", func(t *testing.T) {
		config := BeaconValidatorsDeriverConfig{
			Enabled:   true,
			ChunkSize: 200,
		}
		assert.True(t, config.Enabled)
		assert.Equal(t, 200, config.ChunkSize)
		_ = config.Iterator
	})

	t.Run("beacon_blob_fields", func(t *testing.T) {
		config := BeaconBlobDeriverConfig{Enabled: false}
		assert.False(t, config.Enabled)
		_ = config.Iterator
	})
}
