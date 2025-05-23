package ethereum

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_config",
			config: &Config{
				BeaconNodeAddress: "http://localhost:5052",
			},
			expectError: false,
		},
		{
			name: "valid_config_with_https",
			config: &Config{
				BeaconNodeAddress: "https://beacon.example.com",
			},
			expectError: false,
		},
		{
			name: "missing_beacon_node_address",
			config: &Config{
				BeaconNodeAddress: "",
			},
			expectError: true,
			errorMsg:    "beaconNodeAddress is required",
		},
		{
			name: "valid_config_with_headers",
			config: &Config{
				BeaconNodeAddress: "http://localhost:5052",
				BeaconNodeHeaders: map[string]string{
					"Authorization": "Bearer token123",
				},
			},
			expectError: false,
		},
		{
			name: "valid_config_with_cache_settings",
			config: &Config{
				BeaconNodeAddress: "http://localhost:5052",
				BlockCacheSize:    1000,
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

func TestConfig_DefaultValues(t *testing.T) {
	config := &Config{
		BeaconNodeAddress: "http://localhost:5052",
	}

	err := config.Validate()
	assert.NoError(t, err)

	// Test that config has required field
	assert.NotEmpty(t, config.BeaconNodeAddress)
}

func TestConfig_BeaconNodeHeaders(t *testing.T) {
	config := &Config{
		BeaconNodeAddress: "http://localhost:5052",
		BeaconNodeHeaders: map[string]string{
			"Authorization": "Bearer secret123",
			"X-API-Key":     "key456",
			"User-Agent":    "xatu-cannon/1.0",
		},
	}

	// Test that headers are properly set
	assert.Equal(t, "Bearer secret123", config.BeaconNodeHeaders["Authorization"])
	assert.Equal(t, "key456", config.BeaconNodeHeaders["X-API-Key"])
	assert.Equal(t, "xatu-cannon/1.0", config.BeaconNodeHeaders["User-Agent"])

	// Test validation passes with headers
	err := config.Validate()
	assert.NoError(t, err)
}

func TestConfig_NetworkOverride(t *testing.T) {
	config := &Config{
		BeaconNodeAddress:     "http://localhost:5052",
		OverrideNetworkName:   "mainnet",
	}

	err := config.Validate()
	assert.NoError(t, err)

	assert.Equal(t, "mainnet", config.OverrideNetworkName)
}

func TestConfig_CacheSettings(t *testing.T) {
	config := &Config{
		BeaconNodeAddress: "http://localhost:5052",
		BlockCacheSize:    2000,
	}

	err := config.Validate()
	assert.NoError(t, err)

	assert.Equal(t, uint64(2000), config.BlockCacheSize)
}

func TestConfig_PreloadSettings(t *testing.T) {
	config := &Config{
		BeaconNodeAddress:      "http://localhost:5052",
		BlockPreloadWorkers:    10,
		BlockPreloadQueueSize:  5000,
	}

	err := config.Validate()
	assert.NoError(t, err)

	assert.Equal(t, uint64(10), config.BlockPreloadWorkers)
	assert.Equal(t, uint64(5000), config.BlockPreloadQueueSize)
}

func TestConfig_URLFormats(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "http_localhost",
			url:         "http://localhost:5052",
			expectError: false,
		},
		{
			name:        "https_localhost",
			url:         "https://localhost:5052",
			expectError: false,
		},
		{
			name:        "http_ip",
			url:         "http://127.0.0.1:5052",
			expectError: false,
		},
		{
			name:        "https_domain",
			url:         "https://beacon.example.com",
			expectError: false,
		},
		{
			name:        "https_domain_with_path",
			url:         "https://beacon.example.com/eth/v1",
			expectError: false,
		},
		{
			name:        "invalid_empty_url",
			url:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				BeaconNodeAddress: tt.url,
			}

			err := config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}