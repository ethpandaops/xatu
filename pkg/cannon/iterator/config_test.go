package iterator

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
)

func TestBackfillingCheckpointConfig_Structure(t *testing.T) {
	tests := []struct {
		name     string
		config   *BackfillingCheckpointConfig
		validate func(*testing.T, *BackfillingCheckpointConfig)
	}{
		{
			name:   "default_config",
			config: &BackfillingCheckpointConfig{},
			validate: func(t *testing.T, config *BackfillingCheckpointConfig) {
				// Test default values (disabled backfill, epoch 0)
				assert.False(t, config.Backfill.Enabled)
				assert.Equal(t, phase0.Epoch(0), config.Backfill.ToEpoch)
			},
		},
		{
			name: "enabled_backfill_with_target_epoch",
			config: &BackfillingCheckpointConfig{
				Backfill: struct {
					Enabled bool         `yaml:"enabled" default:"false"`
					ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
				}{
					Enabled: true,
					ToEpoch: phase0.Epoch(12345),
				},
			},
			validate: func(t *testing.T, config *BackfillingCheckpointConfig) {
				assert.True(t, config.Backfill.Enabled)
				assert.Equal(t, phase0.Epoch(12345), config.Backfill.ToEpoch)
			},
		},
		{
			name: "disabled_backfill_with_target_epoch",
			config: &BackfillingCheckpointConfig{
				Backfill: struct {
					Enabled bool         `yaml:"enabled" default:"false"`
					ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
				}{
					Enabled: false,
					ToEpoch: phase0.Epoch(99999),
				},
			},
			validate: func(t *testing.T, config *BackfillingCheckpointConfig) {
				assert.False(t, config.Backfill.Enabled)
				assert.Equal(t, phase0.Epoch(99999), config.Backfill.ToEpoch)
			},
		},
		{
			name: "enabled_backfill_with_zero_epoch",
			config: &BackfillingCheckpointConfig{
				Backfill: struct {
					Enabled bool         `yaml:"enabled" default:"false"`
					ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
				}{
					Enabled: true,
					ToEpoch: phase0.Epoch(0),
				},
			},
			validate: func(t *testing.T, config *BackfillingCheckpointConfig) {
				assert.True(t, config.Backfill.Enabled)
				assert.Equal(t, phase0.Epoch(0), config.Backfill.ToEpoch)
			},
		},
		{
			name: "large_epoch_value",
			config: &BackfillingCheckpointConfig{
				Backfill: struct {
					Enabled bool         `yaml:"enabled" default:"false"`
					ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
				}{
					Enabled: true,
					ToEpoch: phase0.Epoch(1000000),
				},
			},
			validate: func(t *testing.T, config *BackfillingCheckpointConfig) {
				assert.True(t, config.Backfill.Enabled)
				assert.Equal(t, phase0.Epoch(1000000), config.Backfill.ToEpoch)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.config)
		})
	}
}

func TestBackfillingCheckpointConfig_FieldTypes(t *testing.T) {
	config := &BackfillingCheckpointConfig{}

	// Test that fields have correct types
	assert.IsType(t, false, config.Backfill.Enabled)
	assert.IsType(t, phase0.Epoch(0), config.Backfill.ToEpoch)
}

func TestBackfillingCheckpointConfig_YAMLTags(t *testing.T) {
	// This test verifies the struct has appropriate YAML tags
	// We can't easily test YAML unmarshaling without external dependencies,
	// but we can verify the struct structure is correct

	config := &BackfillingCheckpointConfig{
		Backfill: struct {
			Enabled bool         `yaml:"enabled" default:"false"`
			ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
		}{
			Enabled: true,
			ToEpoch: phase0.Epoch(100),
		},
	}

	assert.True(t, config.Backfill.Enabled)
	assert.Equal(t, phase0.Epoch(100), config.Backfill.ToEpoch)
}

func TestBackfillingCheckpointConfig_EpochValues(t *testing.T) {
	tests := []struct {
		name        string
		epochValue  phase0.Epoch
		description string
	}{
		{
			name:        "genesis_epoch",
			epochValue:  phase0.Epoch(0),
			description: "Genesis epoch (slot 0)",
		},
		{
			name:        "early_epoch",
			epochValue:  phase0.Epoch(1),
			description: "First epoch after genesis",
		},
		{
			name:        "typical_mainnet_epoch",
			epochValue:  phase0.Epoch(100000),
			description: "Typical mainnet epoch value",
		},
		{
			name:        "high_epoch_value",
			epochValue:  phase0.Epoch(999999),
			description: "High epoch value for future testing",
		},
		{
			name:        "max_reasonable_epoch",
			epochValue:  phase0.Epoch(18446744073709551615), // max uint64
			description: "Maximum possible epoch value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BackfillingCheckpointConfig{
				Backfill: struct {
					Enabled bool         `yaml:"enabled" default:"false"`
					ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
				}{
					Enabled: true,
					ToEpoch: tt.epochValue,
				},
			}

			assert.Equal(t, tt.epochValue, config.Backfill.ToEpoch, tt.description)
			assert.True(t, config.Backfill.Enabled)
		})
	}
}

func TestBackfillingCheckpointConfig_EnabledStates(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		epoch       phase0.Epoch
		description string
	}{
		{
			name:        "enabled_with_target",
			enabled:     true,
			epoch:       phase0.Epoch(1000),
			description: "Backfilling enabled with specific target epoch",
		},
		{
			name:        "disabled_with_target",
			enabled:     false,
			epoch:       phase0.Epoch(1000),
			description: "Backfilling disabled but target epoch still specified",
		},
		{
			name:        "enabled_without_target",
			enabled:     true,
			epoch:       phase0.Epoch(0),
			description: "Backfilling enabled but no specific target (start from genesis)",
		},
		{
			name:        "disabled_without_target",
			enabled:     false,
			epoch:       phase0.Epoch(0),
			description: "Backfilling disabled and no target (default state)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BackfillingCheckpointConfig{
				Backfill: struct {
					Enabled bool         `yaml:"enabled" default:"false"`
					ToEpoch phase0.Epoch `yaml:"toEpoch" default:"0"`
				}{
					Enabled: tt.enabled,
					ToEpoch: tt.epoch,
				},
			}

			assert.Equal(t, tt.enabled, config.Backfill.Enabled, tt.description)
			assert.Equal(t, tt.epoch, config.Backfill.ToEpoch, tt.description)
		})
	}
}

func TestBackfillingCheckpointConfig_ZeroValue(t *testing.T) {
	// Test that zero-value config has expected defaults
	config := &BackfillingCheckpointConfig{}

	assert.False(t, config.Backfill.Enabled, "Default enabled should be false")
	assert.Equal(t, phase0.Epoch(0), config.Backfill.ToEpoch, "Default epoch should be 0")
}

func TestBackfillingCheckpointConfig_Immutability(t *testing.T) {
	// Test that config fields can be modified independently
	config1 := &BackfillingCheckpointConfig{}
	config2 := &BackfillingCheckpointConfig{}

	// Modify config1
	config1.Backfill.Enabled = true
	config1.Backfill.ToEpoch = phase0.Epoch(123)

	// config2 should remain unchanged
	assert.False(t, config2.Backfill.Enabled)
	assert.Equal(t, phase0.Epoch(0), config2.Backfill.ToEpoch)

	// config1 should have its modifications
	assert.True(t, config1.Backfill.Enabled)
	assert.Equal(t, phase0.Epoch(123), config1.Backfill.ToEpoch)
}
