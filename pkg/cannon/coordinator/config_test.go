package coordinator

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
				Address: "localhost:8080",
				TLS:     false,
			},
			expectError: false,
		},
		{
			name: "valid_config_with_tls",
			config: &Config{
				Address: "secure.example.com:443",
				TLS:     true,
			},
			expectError: false,
		},
		{
			name: "missing_address",
			config: &Config{
				Address: "",
				TLS:     false,
			},
			expectError: true,
			errorMsg:    "address is required",
		},
		{
			name: "valid_config_with_headers",
			config: &Config{
				Address: "localhost:8080",
				TLS:     false,
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"Custom-Header": "value",
				},
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
	// Test that we can create a minimal valid config
	config := &Config{
		Address: "localhost:8080",
	}

	err := config.Validate()
	assert.NoError(t, err, "Minimal config should be valid")

	// Test default TLS value
	assert.False(t, config.TLS, "TLS should default to false")
}

func TestConfig_Headers(t *testing.T) {
	config := &Config{
		Address: "localhost:8080",
		Headers: map[string]string{
			"Authorization": "Bearer secret123",
			"X-API-Key":     "key456", 
			"User-Agent":    "xatu-cannon/1.0",
		},
	}

	// Test that headers are properly accessible
	assert.Equal(t, "Bearer secret123", config.Headers["Authorization"])
	assert.Equal(t, "key456", config.Headers["X-API-Key"])
	assert.Equal(t, "xatu-cannon/1.0", config.Headers["User-Agent"])

	// Test that config is valid with headers
	err := config.Validate()
	assert.NoError(t, err)
}

func TestConfig_AddressFormats(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expectError bool
	}{
		{
			name:        "localhost_with_port",
			address:     "localhost:8080",
			expectError: false,
		},
		{
			name:        "ip_with_port",
			address:     "127.0.0.1:8080",
			expectError: false,
		},
		{
			name:        "domain_with_port",
			address:     "coordinator.example.com:9090",
			expectError: false,
		},
		{
			name:        "ipv6_with_port",
			address:     "[::1]:8080",
			expectError: false,
		},
		{
			name:        "address_without_port",
			address:     "localhost",
			expectError: false, // Validation might not check port format
		},
		{
			name:        "empty_address",
			address:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Address: tt.address,
				TLS:     false,
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