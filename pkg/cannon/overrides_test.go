package cannon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverride_StructureValidation(t *testing.T) {
	tests := []struct {
		name     string
		override *Override
		validate func(*testing.T, *Override)
	}{
		{
			name: "empty_override_with_all_disabled",
			override: &Override{
				MetricsAddr: struct {
					Enabled bool
					Value   string
				}{Enabled: false, Value: ""},
				BeaconNodeURL: struct {
					Enabled bool
					Value   string
				}{Enabled: false, Value: ""},
				BeaconNodeAuthorizationHeader: struct {
					Enabled bool
					Value   string
				}{Enabled: false, Value: ""},
				XatuOutputAuth: struct {
					Enabled bool
					Value   string
				}{Enabled: false, Value: ""},
				XatuCoordinatorAuth: struct {
					Enabled bool
					Value   string
				}{Enabled: false, Value: ""},
				NetworkName: struct {
					Enabled bool
					Value   string
				}{Enabled: false, Value: ""},
			},
			validate: func(t *testing.T, override *Override) {
				assert.False(t, override.MetricsAddr.Enabled)
				assert.False(t, override.BeaconNodeURL.Enabled)
				assert.False(t, override.BeaconNodeAuthorizationHeader.Enabled)
				assert.False(t, override.XatuOutputAuth.Enabled)
				assert.False(t, override.XatuCoordinatorAuth.Enabled)
				assert.False(t, override.NetworkName.Enabled)
			},
		},
		{
			name: "metrics_addr_override_enabled",
			override: &Override{
				MetricsAddr: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   ":8080",
				},
			},
			validate: func(t *testing.T, override *Override) {
				assert.True(t, override.MetricsAddr.Enabled)
				assert.Equal(t, ":8080", override.MetricsAddr.Value)
			},
		},
		{
			name: "beacon_node_url_override_enabled",
			override: &Override{
				BeaconNodeURL: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "http://beacon:5052",
				},
			},
			validate: func(t *testing.T, override *Override) {
				assert.True(t, override.BeaconNodeURL.Enabled)
				assert.Equal(t, "http://beacon:5052", override.BeaconNodeURL.Value)
			},
		},
		{
			name: "beacon_node_auth_header_override_enabled",
			override: &Override{
				BeaconNodeAuthorizationHeader: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "Bearer secret-token",
				},
			},
			validate: func(t *testing.T, override *Override) {
				assert.True(t, override.BeaconNodeAuthorizationHeader.Enabled)
				assert.Equal(t, "Bearer secret-token", override.BeaconNodeAuthorizationHeader.Value)
			},
		},
		{
			name: "xatu_output_auth_override_enabled",
			override: &Override{
				XatuOutputAuth: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "Bearer output-token",
				},
			},
			validate: func(t *testing.T, override *Override) {
				assert.True(t, override.XatuOutputAuth.Enabled)
				assert.Equal(t, "Bearer output-token", override.XatuOutputAuth.Value)
			},
		},
		{
			name: "xatu_coordinator_auth_override_enabled",
			override: &Override{
				XatuCoordinatorAuth: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "Bearer coordinator-token",
				},
			},
			validate: func(t *testing.T, override *Override) {
				assert.True(t, override.XatuCoordinatorAuth.Enabled)
				assert.Equal(t, "Bearer coordinator-token", override.XatuCoordinatorAuth.Value)
			},
		},
		{
			name: "network_name_override_enabled",
			override: &Override{
				NetworkName: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "custom-testnet",
				},
			},
			validate: func(t *testing.T, override *Override) {
				assert.True(t, override.NetworkName.Enabled)
				assert.Equal(t, "custom-testnet", override.NetworkName.Value)
			},
		},
		{
			name: "multiple_overrides_enabled",
			override: &Override{
				MetricsAddr: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   ":9091",
				},
				BeaconNodeURL: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "http://new-beacon:5052",
				},
				NetworkName: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "mainnet",
				},
				XatuCoordinatorAuth: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "Bearer multi-override-token",
				},
			},
			validate: func(t *testing.T, override *Override) {
				assert.True(t, override.MetricsAddr.Enabled)
				assert.Equal(t, ":9091", override.MetricsAddr.Value)

				assert.True(t, override.BeaconNodeURL.Enabled)
				assert.Equal(t, "http://new-beacon:5052", override.BeaconNodeURL.Value)

				assert.True(t, override.NetworkName.Enabled)
				assert.Equal(t, "mainnet", override.NetworkName.Value)

				assert.True(t, override.XatuCoordinatorAuth.Enabled)
				assert.Equal(t, "Bearer multi-override-token", override.XatuCoordinatorAuth.Value)

				// Verify non-set overrides remain disabled
				assert.False(t, override.BeaconNodeAuthorizationHeader.Enabled)
				assert.False(t, override.XatuOutputAuth.Enabled)
			},
		},
		{
			name: "partial_override_enabled_values_only",
			override: &Override{
				MetricsAddr: struct {
					Enabled bool
					Value   string
				}{
					Enabled: false,
					Value:   ":9091", // Value set but not enabled
				},
				BeaconNodeURL: struct {
					Enabled bool
					Value   string
				}{
					Enabled: true,
					Value:   "", // Enabled but empty value
				},
			},
			validate: func(t *testing.T, override *Override) {
				// Disabled override should not apply even with value
				assert.False(t, override.MetricsAddr.Enabled)
				assert.Equal(t, ":9091", override.MetricsAddr.Value)

				// Enabled override should apply even with empty value
				assert.True(t, override.BeaconNodeURL.Enabled)
				assert.Equal(t, "", override.BeaconNodeURL.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.override)
		})
	}
}

func TestOverride_ZeroValue(t *testing.T) {
	// Test that a zero-value Override struct has all fields disabled
	override := &Override{}

	assert.False(t, override.MetricsAddr.Enabled)
	assert.False(t, override.BeaconNodeURL.Enabled)
	assert.False(t, override.BeaconNodeAuthorizationHeader.Enabled)
	assert.False(t, override.XatuOutputAuth.Enabled)
	assert.False(t, override.XatuCoordinatorAuth.Enabled)
	assert.False(t, override.NetworkName.Enabled)

	assert.Empty(t, override.MetricsAddr.Value)
	assert.Empty(t, override.BeaconNodeURL.Value)
	assert.Empty(t, override.BeaconNodeAuthorizationHeader.Value)
	assert.Empty(t, override.XatuOutputAuth.Value)
	assert.Empty(t, override.XatuCoordinatorAuth.Value)
	assert.Empty(t, override.NetworkName.Value)
}

func TestOverride_FieldTypes(t *testing.T) {
	// Test that all fields follow the same pattern
	override := &Override{}

	// Use reflection-like tests to ensure consistent field structure
	tests := []struct {
		name      string
		fieldName string
		testField struct {
			Enabled bool
			Value   string
		}
	}{
		{"MetricsAddr", "MetricsAddr", override.MetricsAddr},
		{"BeaconNodeURL", "BeaconNodeURL", override.BeaconNodeURL},
		{"BeaconNodeAuthorizationHeader", "BeaconNodeAuthorizationHeader", override.BeaconNodeAuthorizationHeader},
		{"XatuOutputAuth", "XatuOutputAuth", override.XatuOutputAuth},
		{"XatuCoordinatorAuth", "XatuCoordinatorAuth", override.XatuCoordinatorAuth},
		{"NetworkName", "NetworkName", override.NetworkName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Each field should be a struct with Enabled bool and Value string
			assert.IsType(t, false, tt.testField.Enabled, "Field %s.Enabled should be bool", tt.fieldName)
			assert.IsType(t, "", tt.testField.Value, "Field %s.Value should be string", tt.fieldName)
		})
	}
}

func TestOverride_UsagePatterns(t *testing.T) {
	t.Run("typical_production_override", func(t *testing.T) {
		// Simulate a typical production override scenario
		override := &Override{
			BeaconNodeURL: struct {
				Enabled bool
				Value   string
			}{
				Enabled: true,
				Value:   "https://prod-beacon.example.com:5052",
			},
			BeaconNodeAuthorizationHeader: struct {
				Enabled bool
				Value   string
			}{
				Enabled: true,
				Value:   "Bearer prod-api-key",
			},
			NetworkName: struct {
				Enabled bool
				Value   string
			}{
				Enabled: true,
				Value:   "mainnet",
			},
			MetricsAddr: struct {
				Enabled bool
				Value   string
			}{
				Enabled: true,
				Value:   "0.0.0.0:9090",
			},
		}

		// Verify production override settings
		assert.True(t, override.BeaconNodeURL.Enabled)
		assert.Contains(t, override.BeaconNodeURL.Value, "https://")
		assert.Contains(t, override.BeaconNodeURL.Value, ":5052")

		assert.True(t, override.BeaconNodeAuthorizationHeader.Enabled)
		assert.Contains(t, override.BeaconNodeAuthorizationHeader.Value, "Bearer")

		assert.True(t, override.NetworkName.Enabled)
		assert.Equal(t, "mainnet", override.NetworkName.Value)

		assert.True(t, override.MetricsAddr.Enabled)
		assert.Contains(t, override.MetricsAddr.Value, "9090")
	})

	t.Run("typical_development_override", func(t *testing.T) {
		// Simulate a typical development override scenario
		override := &Override{
			BeaconNodeURL: struct {
				Enabled bool
				Value   string
			}{
				Enabled: true,
				Value:   "http://localhost:5052",
			},
			NetworkName: struct {
				Enabled bool
				Value   string
			}{
				Enabled: true,
				Value:   "sepolia",
			},
		}

		// Verify development override settings
		assert.True(t, override.BeaconNodeURL.Enabled)
		assert.Contains(t, override.BeaconNodeURL.Value, "localhost")

		assert.True(t, override.NetworkName.Enabled)
		assert.Equal(t, "sepolia", override.NetworkName.Value)

		// Other overrides should be disabled in development
		assert.False(t, override.BeaconNodeAuthorizationHeader.Enabled)
		assert.False(t, override.XatuOutputAuth.Enabled)
		assert.False(t, override.XatuCoordinatorAuth.Enabled)
		assert.False(t, override.MetricsAddr.Enabled)
	})
}
