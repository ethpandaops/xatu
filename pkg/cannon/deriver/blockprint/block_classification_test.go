package blockprint

import (
	"context"
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBlockClassificationDeriver_Name(t *testing.T) {
	config := &BlockClassificationDeriverConfig{
		Enabled:   true,
		BatchSize: 10,
		Endpoint:  "http://localhost:8080",
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &BlockClassificationDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	assert.Equal(t, BlockClassificationName.String(), deriver.Name())
}

func TestBlockClassificationDeriver_CannonType(t *testing.T) {
	config := &BlockClassificationDeriverConfig{
		Enabled:   true,
		BatchSize: 10,
		Endpoint:  "http://localhost:8080",
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &BlockClassificationDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	assert.Equal(t, BlockClassificationName, deriver.CannonType())
}

func TestBlockClassificationDeriver_ActivationFork(t *testing.T) {
	config := &BlockClassificationDeriverConfig{
		Enabled:   true,
		BatchSize: 10,
		Endpoint:  "http://localhost:8080",
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &BlockClassificationDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	// Test that it returns a valid fork version
	fork := deriver.ActivationFork()
	assert.True(t, fork == spec.DataVersionPhase0 || 
		fork == spec.DataVersionAltair || 
		fork == spec.DataVersionBellatrix ||
		fork == spec.DataVersionCapella ||
		fork == spec.DataVersionDeneb)
}

func TestBlockClassificationDeriverConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *BlockClassificationDeriverConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_config",
			config: &BlockClassificationDeriverConfig{
				Enabled:   true,
				BatchSize: 10,
				Endpoint:  "http://localhost:8080",
			},
			expectError: false,
		},
		{
			name: "invalid_batch_size_zero",
			config: &BlockClassificationDeriverConfig{
				Enabled:   true,
				BatchSize: 0,
				Endpoint:  "http://localhost:8080",
			},
			expectError: true,
			errorMsg:    "batch size must be greater than 0",
		},
		{
			name: "invalid_batch_size_negative",
			config: &BlockClassificationDeriverConfig{
				Enabled:   true,
				BatchSize: -1,
				Endpoint:  "http://localhost:8080",
			},
			expectError: true,
			errorMsg:    "batch size must be greater than 0",
		},
		{
			name: "valid_disabled_config",
			config: &BlockClassificationDeriverConfig{
				Enabled:   false,
				BatchSize: 1, // Still needs to be valid even when disabled
				Endpoint:  "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBlockClassificationDeriver_OnEventsDerived(t *testing.T) {
	config := &BlockClassificationDeriverConfig{
		Enabled:   true,
		BatchSize: 10,
		Endpoint:  "http://localhost:8080",
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &BlockClassificationDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	// Test callback registration
	callbackCount := 0
	callback := func(ctx context.Context, events []*xatu.DecoratedEvent) error {
		callbackCount++
		return nil
	}

	// Initially no callbacks
	assert.Len(t, deriver.onEventsCallbacks, 0)

	// Register callback
	deriver.OnEventsDerived(context.Background(), callback)

	// Verify callback was registered
	assert.Len(t, deriver.onEventsCallbacks, 1)

	// Register another callback
	deriver.OnEventsDerived(context.Background(), callback)
	assert.Len(t, deriver.onEventsCallbacks, 2)
}

func TestNewBlockClassificationDeriver(t *testing.T) {
	config := &BlockClassificationDeriverConfig{
		Enabled:   true,
		BatchSize: 10,
		Endpoint:  "http://localhost:8080",
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	log := logrus.NewEntry(logrus.New())

	// Test constructor (with nil dependencies for unit test)
	deriver := NewBlockClassificationDeriver(log, config, nil, nil, clientMeta, nil)

	assert.NotNil(t, deriver)
	assert.Equal(t, config, deriver.cfg)
	assert.Equal(t, clientMeta, deriver.clientMeta)
	assert.NotNil(t, deriver.log)
	assert.Len(t, deriver.onEventsCallbacks, 0)
}

func TestBlockClassificationDeriver_ImplementsEventDeriver(t *testing.T) {
	config := &BlockClassificationDeriverConfig{
		Enabled:   true,
		BatchSize: 10,
		Endpoint:  "http://localhost:8080",
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &BlockClassificationDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	// Test interface methods exist and return expected types
	assert.IsType(t, "", deriver.Name())
	assert.IsType(t, xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION, deriver.CannonType())
	assert.IsType(t, spec.DataVersionPhase0, deriver.ActivationFork())

	// Test that interface methods exist
	// Note: We don't test Start() with nil dependencies as it would panic
	// That's tested in integration tests with proper mocks

	// Stop should work even with nil dependencies
	err := deriver.Stop(context.Background())
	assert.NoError(t, err)
}

func TestBlockClassificationDeriver_Constants(t *testing.T) {
	// Test that constants are properly defined
	assert.Equal(t, xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION, BlockClassificationName)
	assert.NotEmpty(t, BlockClassificationName.String())
}

func TestBlockClassificationDeriverConfig_DefaultValues(t *testing.T) {
	// Test that we can work with default configurations
	config := &BlockClassificationDeriverConfig{}

	// Should fail validation with default values
	err := config.Validate()
	assert.Error(t, err, "Default config should fail validation due to zero batch size")

	// Test with minimal valid config
	config.BatchSize = 1
	err = config.Validate()
	assert.NoError(t, err, "Config with valid batch size should pass validation")
}

func TestBlockClassificationDeriver_ConfigFields(t *testing.T) {
	config := &BlockClassificationDeriverConfig{
		Enabled:   true,
		Endpoint:  "http://example.com:8080",
		BatchSize: 50,
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"Custom-Header": "value",
		},
	}

	// Test that all fields are properly accessible
	assert.True(t, config.Enabled)
	assert.Equal(t, "http://example.com:8080", config.Endpoint)
	assert.Equal(t, 50, config.BatchSize)
	assert.Equal(t, "Bearer token123", config.Headers["Authorization"])
	assert.Equal(t, "value", config.Headers["Custom-Header"])

	// Test validation passes for complete config
	err := config.Validate()
	assert.NoError(t, err)
}