package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ShouldFailOnMissingTables(t *testing.T) {
	tests := []struct {
		name     string
		field    *bool
		expected bool
	}{
		{name: "nil_defaults_to_true", field: nil, expected: true},
		{name: "explicit_true", field: boolPtr(true), expected: true},
		{name: "explicit_false_survives_default", field: boolPtr(false), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{FailOnMissingTables: tt.field}
			assert.Equal(t, tt.expected, c.ShouldFailOnMissingTables())
		})
	}
}

func TestAdaptiveLimiterConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		field    *bool
		expected bool
	}{
		{name: "nil_defaults_to_true", field: nil, expected: true},
		{name: "explicit_true", field: boolPtr(true), expected: true},
		{name: "explicit_false_survives_default", field: boolPtr(false), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AdaptiveLimiterConfig{Enabled: tt.field}
			assert.Equal(t, tt.expected, c.IsEnabled())
		})
	}
}
