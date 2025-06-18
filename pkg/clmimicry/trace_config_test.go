package clmimicry

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindMatchingTopicConfig validates the pattern matching logic that maps
// incoming libp2p trace event types to their configured sharding rules.
//
// This test ensures the regex compilation and matching system correctly:
// 1. Matches exact event type names
// 2. Handles case-insensitive patterns
// 3. Supports partial pattern matching with wildcards
// 4. Returns appropriate results for non-matching patterns
//
// Pattern matching is fundamental to the sharding system - events must be
// correctly mapped to their configuration to apply proper sampling rates.
func TestFindMatchingTopicConfig(t *testing.T) {
	uint64Ptr := uint64(64)

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
						TotalShards:     &uint64Ptr,
						ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
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
						TotalShards:     &uint64Ptr,
						ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
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
						TotalShards:     &uint64Ptr,
						ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
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
						TotalShards:     &uint64Ptr,
						ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
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
			// Validate and pre-compile patterns for testing.
			if tt.config.Enabled && len(tt.config.Topics) > 0 {
				err := tt.config.Validate()
				require.NoError(t, err)
			}
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

func TestLogSummary(t *testing.T) {
	uint64Ptr64 := uint64(64)
	uint64Ptr1 := uint64(1)

	tests := []struct {
		name               string
		config             TracesConfig
		expectedPhrases    []string
		notExpectedPhrases []string
	}{
		{
			name: "disabled config",
			config: TracesConfig{
				Enabled: false,
				Topics: map[string]TopicConfig{
					"beacon_block": {
						TotalShards:     &uint64Ptr64,
						ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
					},
				},
			},
			expectedPhrases: []string{
				"Trace-based sampling disabled",
			},
			notExpectedPhrases: []string{
				"beacon_block",
			},
		},
		{
			name: "enabled but empty topics",
			config: TracesConfig{
				Enabled: true,
				Topics:  map[string]TopicConfig{},
			},
			expectedPhrases: []string{
				"Trace-based sampling enabled but no topics configured",
			},
		},
		{
			name: "regular config",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"beacon_block": {
						TotalShards:     &uint64Ptr64,
						ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
					},
					"beacon_attestation": {
						TotalShards:     &uint64Ptr64,
						ActiveShardsRaw: &ActiveShardsConfig{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
					},
				},
			},
			expectedPhrases: []string{
				"Trace-based sampling enabled with 2 topic patterns",
				"Pattern 'beacon_block': 3/64 shards active (4.7%)",
				"Pattern 'beacon_attestation': 16/64 shards active (25.0%)",
				"[0,1,2,3,4,5,6,7,8,9",
			},
		},
		{
			name: "firehose config",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"blob_sidecar": {
						TotalShards:     &uint64Ptr1,
						ActiveShardsRaw: &ActiveShardsConfig{0},
					},
				},
			},
			expectedPhrases: []string{
				"Pattern 'blob_sidecar': FIREHOSE (all 1 shards active)",
			},
			notExpectedPhrases: []string{
				"shards active (100.0%)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the config to process ActiveShardsRaw into ActiveShards
			if tt.config.Enabled && len(tt.config.Topics) > 0 {
				err := tt.config.Validate()
				require.NoError(t, err)
			}

			summary := tt.config.LogSummary()

			for _, phrase := range tt.expectedPhrases {
				assert.True(t, strings.Contains(summary, phrase),
					"Expected summary to contain phrase: %s, but got: %s", phrase, summary)
			}

			for _, phrase := range tt.notExpectedPhrases {
				assert.False(t, strings.Contains(summary, phrase),
					"Expected summary to NOT contain phrase: %s, but got: %s", phrase, summary)
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
	uint64Ptr := uint64(64)

	config := TracesConfig{
		Enabled: true,
		Topics: map[string]TopicConfig{
			"beacon_block":   {TotalShards: &uint64Ptr, ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3}},
			"beacon_.*":      {TotalShards: &uint64Ptr, ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3}},
			"sync_committee": {TotalShards: &uint64Ptr, ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3}},
			"blob_.*":        {TotalShards: &uint64Ptr, ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3}},
			"[a-z]+_[0-9]+":  {TotalShards: &uint64Ptr, ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3}},
		},
	}

	// Validate and pre-compile patterns for optimized version
	err := config.Validate()
	if err != nil {
		b.Fatalf("Failed to validate config: %v", err)
	}

	err = config.CompilePatterns()
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

func TestActiveShardsConfig_ToUint64Slice(t *testing.T) {
	tests := []struct {
		name     string
		input    ActiveShardsConfig
		expected []uint64
		wantErr  bool
	}{
		{
			name:     "individual numbers",
			input:    ActiveShardsConfig{0, 1, 2, 3},
			expected: []uint64{0, 1, 2, 3},
			wantErr:  false,
		},
		{
			name:     "range syntax",
			input:    ActiveShardsConfig{"0-3"},
			expected: []uint64{0, 1, 2, 3},
			wantErr:  false,
		},
		{
			name:     "mixed individual and range",
			input:    ActiveShardsConfig{0, "2-4", 10},
			expected: []uint64{0, 2, 3, 4, 10},
			wantErr:  false,
		},
		{
			name:  "larger range",
			input: ActiveShardsConfig{"0-255"},
			expected: func() []uint64 {
				var result []uint64

				for i := uint64(0); i <= 255; i++ {
					result = append(result, i)
				}

				return result
			}(),
			wantErr: false,
		},
		{
			name:     "duplicates removed",
			input:    ActiveShardsConfig{0, 1, "0-2", 2},
			expected: []uint64{0, 1, 2},
			wantErr:  false,
		},
		{
			name:     "invalid range format",
			input:    ActiveShardsConfig{"0-1-2"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid range start",
			input:    ActiveShardsConfig{"a-5"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid range end",
			input:    ActiveShardsConfig{"0-b"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "start greater than end",
			input:    ActiveShardsConfig{"5-2"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "string numbers",
			input:    ActiveShardsConfig{"0", "1", "2"},
			expected: []uint64{0, 1, 2},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.input.ToUint64Slice()

			if (err != nil) != tt.wantErr {
				t.Errorf("ToUint64Slice() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				if len(result) != len(tt.expected) {
					t.Errorf("ToUint64Slice() result length = %d, expected %d", len(result), len(tt.expected))

					return
				}

				for i, v := range result {
					if v != tt.expected[i] {
						t.Errorf("ToUint64Slice() result[%d] = %d, expected %d", i, v, tt.expected[i])
					}
				}
			}
		})
	}
}

func TestTracesConfig_ValidateWithRanges(t *testing.T) {
	totalShards := uint64(10)
	activeShards := ActiveShardsConfig{"0-4", 7, "8-9"}

	config := &TracesConfig{
		Enabled: true,
		Topics: map[string]TopicConfig{
			"test_pattern": {
				TotalShards:     &totalShards,
				ActiveShardsRaw: &activeShards,
				ShardingKey:     "MsgID",
			},
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Check that ranges were expanded correctly.
	topicConfig := config.Topics["test_pattern"]
	expected := []uint64{0, 1, 2, 3, 4, 7, 8, 9}

	if len(topicConfig.ActiveShards) != len(expected) {
		t.Fatalf("Expected %d active shards, got %d", len(expected), len(topicConfig.ActiveShards))
	}

	for i, v := range topicConfig.ActiveShards {
		if v != expected[i] {
			t.Errorf("ActiveShards[%d] = %d, expected %d", i, v, expected[i])
		}
	}
}
