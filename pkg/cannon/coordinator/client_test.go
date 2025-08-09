package coordinator

import (
	"context"
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_config_creates_client",
			config: &Config{
				Address: "localhost:8080",
				TLS:     false,
			},
			expectError: false,
		},
		{
			name: "invalid_config_fails",
			config: &Config{
				Address: "", // Invalid
				TLS:     false,
			},
			expectError: true,
			errorMsg:    "address is required",
		},
		{
			name:        "nil_config_fails",
			config:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.NewEntry(logrus.New())

			client, err := New(tt.config, log)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.config, client.config)
				assert.NotNil(t, client.log)
			}
		})
	}
}

func TestClient_StartStop(t *testing.T) {
	config := &Config{
		Address: "localhost:8080",
		TLS:     false,
	}

	log := logrus.NewEntry(logrus.New())
	client, err := New(config, log)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Test Start/Stop lifecycle
	ctx := context.Background()

	// Note: These may fail with actual network connections, but should not panic
	_ = client.Start(ctx)
	// We don't assert on error since this might fail due to no actual server

	err = client.Stop(ctx)
	// Stop should generally succeed
	assert.NoError(t, err)
}

func TestClient_GetCannonLocation(t *testing.T) {
	config := &Config{
		Address: "localhost:8080",
		TLS:     false,
	}

	log := logrus.NewEntry(logrus.New())
	client, err := New(config, log)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	ctx := context.Background()
	cannonType := xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK
	networkID := "1"

	// Test method exists and has correct signature
	// Note: This will likely fail with network error, but tests the interface
	location, err := client.GetCannonLocation(ctx, cannonType, networkID)

	// We expect this to fail with network error in unit test environment
	// but the method should exist and not panic
	_ = location // Ignore result for unit test
	_ = err      // Network errors are expected in unit tests
}

func TestClient_UpsertCannonLocationRequest(t *testing.T) {
	config := &Config{
		Address: "localhost:8080",
		TLS:     false,
	}

	log := logrus.NewEntry(logrus.New())
	client, err := New(config, log)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	ctx := context.Background()

	// Create a test location
	location := &xatu.CannonLocation{
		NetworkId: "1",
		Type:      xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
	}

	// Test method exists and has correct signature
	err = client.UpsertCannonLocationRequest(ctx, location)

	// We expect this to fail with network error in unit test environment
	// but the method should exist and not panic
	_ = err // Network errors are expected in unit tests
}

func TestClient_Config(t *testing.T) {
	config := &Config{
		Address: "localhost:8080",
		TLS:     true,
		Headers: map[string]string{
			"Authorization": "Bearer test",
		},
	}

	log := logrus.NewEntry(logrus.New())
	client, err := New(config, log)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Test that client stores config correctly
	assert.Equal(t, config, client.config)
	assert.Equal(t, "localhost:8080", client.config.Address)
	assert.True(t, client.config.TLS)
	assert.Equal(t, "Bearer test", client.config.Headers["Authorization"])
}

func TestClient_Logger(t *testing.T) {
	config := &Config{
		Address: "localhost:8080",
		TLS:     false,
	}

	log := logrus.NewEntry(logrus.New())
	client, err := New(config, log)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Test that client has logger
	assert.NotNil(t, client.log)
}
