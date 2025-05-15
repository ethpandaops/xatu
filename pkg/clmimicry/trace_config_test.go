package clmimicry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindMatchingTopicConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    TracesConfig
		eventType string
		wantMatch bool
	}{
		{
			name: "exact match",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"beacon_block": {
						TotalShards:  64,
						ActiveShards: []uint64{1, 2, 3},
					},
				},
			},
			eventType: "beacon_block",
			wantMatch: true,
		},
		{
			name: "regex match",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					".*beacon_attestation.*": {
						TotalShards:  64,
						ActiveShards: []uint64{1, 2, 3},
					},
				},
			},
			eventType: "beacon_attestation",
			wantMatch: true,
		},
		{
			name: "no match",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"beacon_block": {
						TotalShards:  64,
						ActiveShards: []uint64{1, 2, 3},
					},
				},
			},
			eventType: "other_event",
			wantMatch: false,
		},
		{
			name: "disabled config",
			config: TracesConfig{
				Enabled: false,
				Topics: map[string]TopicConfig{
					"beacon_block": {
						TotalShards:  64,
						ActiveShards: []uint64{1, 2, 3},
					},
				},
			},
			eventType: "beacon_block",
			wantMatch: false,
		},
		{
			name: "empty topics",
			config: TracesConfig{
				Enabled: true,
				Topics:  map[string]TopicConfig{},
			},
			eventType: "beacon_block",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pre-compile patterns for testing.
			err := tt.config.CompilePatterns()
			require.NoError(t, err)

			config, found := tt.config.FindMatchingTopicConfig(tt.eventType)
			assert.Equal(t, tt.wantMatch, found)

			if tt.wantMatch {
				assert.NotNil(t, config)
			} else {
				assert.Nil(t, config)
			}
		})
	}
}

func TestCompilePatterns(t *testing.T) {
	tests := []struct {
		name      string
		config    TracesConfig
		wantError bool
	}{
		{
			name: "valid patterns",
			config: TracesConfig{
				Topics: map[string]TopicConfig{
					"beacon_block":  {},
					"beacon_.*":     {},
					"[a-z]+_[0-9]+": {},
				},
			},
			wantError: false,
		},
		{
			name: "invalid pattern",
			config: TracesConfig{
				Topics: map[string]TopicConfig{
					"beacon_block": {},
					"[unclosed":    {},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.CompilePatterns()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.config.Topics), len(tt.config.compiledPatterns))

				for pattern := range tt.config.Topics {
					assert.Contains(t, tt.config.compiledPatterns, pattern)
				}
			}
		})
	}
}

func BenchmarkFindMatchingTopicConfig(b *testing.B) {
	config := TracesConfig{
		Enabled: true,
		Topics: map[string]TopicConfig{
			"beacon_block":   {TotalShards: 64, ActiveShards: []uint64{1, 2, 3}},
			"beacon_.*":      {TotalShards: 64, ActiveShards: []uint64{1, 2, 3}},
			"sync_committee": {TotalShards: 64, ActiveShards: []uint64{1, 2, 3}},
			"blob_.*":        {TotalShards: 64, ActiveShards: []uint64{1, 2, 3}},
			"[a-z]+_[0-9]+":  {TotalShards: 64, ActiveShards: []uint64{1, 2, 3}},
		},
	}

	// Pre-compile patterns for optimized version
	err := config.CompilePatterns()
	if err != nil {
		b.Fatalf("Failed to compile patterns: %v", err)
	}

	eventTypes := []string{
		"beacon_block",
		"beacon_attestation",
		"sync_committee",
		"blob_sidecar",
		"test_123",
		"no_match_event",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		eventType := eventTypes[i%len(eventTypes)]
		config.FindMatchingTopicConfig(eventType)
	}
}
