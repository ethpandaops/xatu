package clmimicry

import (
	"fmt"
	"math"
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfigurableTotalShards(t *testing.T) {
	// Test configuration with different totalShards values
	config := &ShardingConfigV2{
		Topics: map[string]*TopicShardingConfig{
			"small_shards": {
				TotalShards:  256,
				ActiveShards: []uint64{0, 1, 2, 3}, // 4/256 = 1.56% sampling
			},
			"standard_shards": {
				TotalShards:  512,
				ActiveShards: []uint64{0, 1, 2, 3}, // 4/512 = 0.78% sampling
			},
			"large_shards": {
				TotalShards:  1024,
				ActiveShards: []uint64{0, 1, 2, 3}, // 4/1024 = 0.39% sampling
			},
		},
	}

	// Validate configuration
	err := config.validate()
	require.NoError(t, err)

	// Test sampling rates
	assert.InDelta(t, 0.0156, config.Topics["small_shards"].GetSamplingRate(), 0.001)
	assert.InDelta(t, 0.0078, config.Topics["standard_shards"].GetSamplingRate(), 0.001)
	assert.InDelta(t, 0.0039, config.Topics["large_shards"].GetSamplingRate(), 0.001)

	// Test that sharding works with different totalShards
	_, err = NewUnifiedSharder(config, true)
	require.NoError(t, err)

	// Test that the same key produces different shards with different totalShards
	key := "test_message_id"
	shard256 := GetShard(key, 256)
	shard512 := GetShard(key, 512)
	shard1024 := GetShard(key, 1024)

	// Verify shards are within their respective ranges
	assert.Less(t, shard256, uint64(256))
	assert.Less(t, shard512, uint64(512))
	assert.Less(t, shard1024, uint64(1024))

	t.Logf("Key '%s' maps to: shard %d (of 256), shard %d (of 512), shard %d (of 1024)",
		key, shard256, shard512, shard1024)
}

func TestValidateTotalShards(t *testing.T) {
	tests := []struct {
		name    string
		config  *TopicShardingConfig
		pattern string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &TopicShardingConfig{
				TotalShards:  512,
				ActiveShards: []uint64{0, 1, 2},
			},
			pattern: "test_pattern",
			wantErr: false,
		},
		{
			name: "zero totalShards",
			config: &TopicShardingConfig{
				TotalShards:  0,
				ActiveShards: []uint64{0, 1, 2},
			},
			pattern: "test_pattern",
			wantErr: true,
			errMsg:  "totalShards must be greater than 0",
		},
		{
			name: "shard out of range",
			config: &TopicShardingConfig{
				TotalShards:  256,
				ActiveShards: []uint64{0, 1, 256}, // 256 is out of range (0-255)
			},
			pattern: "test_pattern",
			wantErr: true,
			errMsg:  "active shard 256 is out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate(tt.pattern)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUnifiedSharder(t *testing.T) {
	tests := []struct {
		name           string
		config         *ShardingConfigV2
		eventType      xatu.Event_Name
		msgID          string
		topic          string
		expectedResult bool
		expectedReason string
	}{
		{
			name: "Group A event with topic match",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					".*beacon_block.*": {
						TotalShards:  512,
						ActiveShards: []uint64{0, 1, 2, 3, 4}, // 5 shards
					},
				},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			msgID:          "test-msg-id",
			topic:          "beacon_block",
			expectedResult: false, // Depends on hash, but likely false with only 5 shards
		},
		{
			name: "Group A event without topic config falls back to msgID",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
			msgID:          "test-msg-id",
			topic:          "unknown_topic",
			expectedResult: false, // Default to shard 0 only
			expectedReason: "group_a_no_topic_config",
		},
		{
			name: "Group B event with topic match",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					".*attestation.*": {
						TotalShards:  512,
						ActiveShards: generateShardRange(0, 255), // 50% sampling
					},
				},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_JOIN,
			msgID:          "",
			topic:          "beacon_attestation_1",
			expectedResult: true, // 50% chance
		},
		{
			name: "Group B event without topic config",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_LEAVE,
			msgID:          "",
			topic:          "some_topic",
			expectedResult: false,
			expectedReason: "group_b_no_config",
		},
		{
			name: "Group C event with msgID",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
			msgID:          "test-msg-id",
			topic:          "",
			expectedResult: false, // Default to shard 0 only
		},
		{
			name: "Group D event with enabled config",
			config: &ShardingConfigV2{
				NoShardingKeyEvents: &NoShardingKeyConfig{
					Enabled: true,
				},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_ADD_PEER,
			msgID:          "",
			topic:          "",
			expectedResult: true,
			expectedReason: "group_d_enabled",
		},
		{
			name: "Group D event with disabled config",
			config: &ShardingConfigV2{
				NoShardingKeyEvents: &NoShardingKeyConfig{
					Enabled: false,
				},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
			msgID:          "",
			topic:          "",
			expectedResult: false,
			expectedReason: "group_d_disabled",
		},
		{
			name: "Highest sampling rate selection",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					".*beacon.*": {
						TotalShards:  512,
						ActiveShards: generateShardRange(0, 127), // 25% sampling
					},
					".*beacon_block.*": {
						TotalShards:  512,
						ActiveShards: generateShardRange(0, 511), // 100% sampling
					},
				},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			msgID:          "test-msg-id",
			topic:          "beacon_block",
			expectedResult: true, // Should use 100% sampling config
		},
		{
			name: "Sharding disabled",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					".*": {
						TotalShards:  512,
						ActiveShards: []uint64{0}, // Minimal sampling
					},
				},
			},
			eventType:      xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
			msgID:          "test-msg-id",
			topic:          "any_topic",
			expectedResult: true, // When sharder is created with enabled=false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile patterns
			err := tt.config.compilePatterns()
			require.NoError(t, err)

			// Create sharder
			sharder, err := NewUnifiedSharder(tt.config, tt.name != "Sharding disabled")
			require.NoError(t, err)

			// Test sharding decision
			result, reason := sharder.ShouldProcess(tt.eventType, tt.msgID, tt.topic)

			if tt.expectedReason != "" {
				assert.Equal(t, tt.expectedReason, reason, "Unexpected reason")
			}

			// For deterministic tests
			if tt.name == "Group D event with enabled config" ||
				tt.name == "Group D event with disabled config" ||
				tt.name == "Sharding disabled" {
				assert.Equal(t, tt.expectedResult, result, "Unexpected result")
			}
		})
	}
}

func TestShardDistribution(t *testing.T) {
	config := &ShardingConfigV2{
		Topics: map[string]*TopicShardingConfig{
			".*": {
				TotalShards:  512,
				ActiveShards: generateShardRange(0, 511), // All shards
			},
		},
	}

	err := config.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(config, true)
	require.NoError(t, err)

	// Test shard distribution using chi-squared test
	totalShards := uint64(512)
	numTests := 100000 // Large sample size for statistical significance
	shardCounts := make([]int, totalShards)

	for i := 0; i < numTests; i++ {
		msgID := fmt.Sprintf("test-message-%d", i)
		shard := sharder.GetShardForKey(msgID, totalShards)
		shardCounts[shard]++
	}

	// Calculate chi-squared statistic
	expectedPerShard := float64(numTests) / float64(totalShards)
	chiSquared := 0.0
	minCount := numTests
	maxCount := 0

	for _, count := range shardCounts {
		if count < minCount {
			minCount = count
		}
		if count > maxCount {
			maxCount = count
		}
		diff := float64(count) - expectedPerShard
		chiSquared += (diff * diff) / expectedPerShard
	}

	// For 511 degrees of freedom (512 shards - 1), the critical value
	// at 0.05 significance level is approximately 564.7
	// We'll use a more conservative threshold of 600
	criticalValue := 600.0

	t.Logf("Chi-squared value: %.2f (critical value: %.2f)", chiSquared, criticalValue)
	t.Logf("Min count: %d, Max count: %d, Expected: %.1f", minCount, maxCount, expectedPerShard)
	t.Logf("Range spread: %.1f%% of expected", float64(maxCount-minCount)/expectedPerShard*100)

	assert.Less(t, chiSquared, criticalValue,
		"Distribution is not uniform (chi-squared: %.2f > %.2f)", chiSquared, criticalValue)

	// Check that most shards are within expected bounds
	// Using 3-sigma rule: 99.7% should be within 3 standard deviations
	// For binomial distribution: stddev = sqrt(n * p * (1-p))
	// where p = 1/512 for uniform distribution
	p := 1.0 / float64(totalShards)
	stdDev := math.Sqrt(float64(numTests) * p * (1 - p))

	// Count how many shards fall outside different sigma levels
	outsideOneSigma := 0
	outsideTwoSigma := 0
	outsideThreeSigma := 0
	outsideFourSigma := 0

	for _, count := range shardCounts {
		deviation := abs(float64(count) - expectedPerShard)
		if deviation > stdDev {
			outsideOneSigma++
		}
		if deviation > 2*stdDev {
			outsideTwoSigma++
		}
		if deviation > 3*stdDev {
			outsideThreeSigma++
		}
		if deviation > 4*stdDev {
			outsideFourSigma++
		}
	}

	// Expected percentages for normal distribution
	// Outside 1σ: ~31.7% (162 shards)
	// Outside 2σ: ~4.6% (24 shards)
	// Outside 3σ: ~0.3% (1-2 shards)
	// Outside 4σ: ~0.006% (0 shards)

	t.Logf("Shards outside 1σ: %d (%.1f%%), expected ~31.7%%",
		outsideOneSigma, float64(outsideOneSigma)/float64(totalShards)*100)
	t.Logf("Shards outside 2σ: %d (%.1f%%), expected ~4.6%%",
		outsideTwoSigma, float64(outsideTwoSigma)/float64(totalShards)*100)
	t.Logf("Shards outside 3σ: %d (%.1f%%), expected ~0.3%%",
		outsideThreeSigma, float64(outsideThreeSigma)/float64(totalShards)*100)
	t.Logf("Shards outside 4σ: %d (%.1f%%), expected ~0.006%%",
		outsideFourSigma, float64(outsideFourSigma)/float64(totalShards)*100)

	// Allow some tolerance for statistical variance
	// We expect about 0.3% (1-2 shards) outside 3σ, so allow up to 5 shards (1%)
	assert.LessOrEqual(t, outsideThreeSigma, 5,
		"Too many shards outside 3σ: %d > 5 (%.1f%% > 1%%)",
		outsideThreeSigma, float64(outsideThreeSigma)/float64(totalShards)*100)

	// Should have essentially no shards outside 4σ
	assert.LessOrEqual(t, outsideFourSigma, 1,
		"Too many shards outside 4σ: %d > 1", outsideFourSigma)
}

func TestEventCategorizer(t *testing.T) {
	ec := NewEventCategorizer()

	tests := []struct {
		eventType     xatu.Event_Name
		expectedGroup ShardingGroup
		hasTopic      bool
		hasMsgID      bool
	}{
		// Group A tests
		{xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE, GroupA, true, true},
		{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK, GroupA, true, true},
		{xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE, GroupA, true, true},
		{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE, GroupA, true, true},

		// Group B tests
		{xatu.Event_LIBP2P_TRACE_JOIN, GroupB, true, false},
		{xatu.Event_LIBP2P_TRACE_LEAVE, GroupB, true, false},
		{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT, GroupB, true, false},

		// Group C tests
		{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT, GroupC, false, true},
		{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT, GroupC, false, true},

		// Group D tests
		{xatu.Event_LIBP2P_TRACE_ADD_PEER, GroupD, false, false},
		{xatu.Event_LIBP2P_TRACE_CONNECTED, GroupD, false, false},
		{xatu.Event_LIBP2P_TRACE_RECV_RPC, GroupD, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType.String(), func(t *testing.T) {
			info, exists := ec.GetEventInfo(tt.eventType)
			assert.True(t, exists, "Event %s should exist", tt.eventType)
			assert.Equal(t, tt.expectedGroup, info.ShardingGroup, "Wrong sharding group")
			assert.Equal(t, tt.hasTopic, info.HasTopic, "Wrong HasTopic")
			assert.Equal(t, tt.hasMsgID, info.HasMsgID, "Wrong HasMsgID")
		})
	}

	// Test unknown event
	unknownEvent := xatu.Event_Name(9999)
	group := ec.GetShardingGroup(unknownEvent)
	assert.Equal(t, GroupD, group, "Unknown events should default to GroupD")
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *ShardingConfigV2
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config with ranges",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					".*": {
						TotalShards:  512,
						ActiveShards: generateShardRange(0, 255), // Valid range
					},
				},
			},
			expectError: false,
		},
		{
			name: "Invalid shard out of range",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					".*": {
						TotalShards:  512,
						ActiveShards: []uint64{512}, // Out of range
					},
				},
			},
			expectError: true,
			errorMsg:    "out of range",
		},
		{
			name: "Empty active shards",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					".*": {
						TotalShards:  512,
						ActiveShards: []uint64{},
					},
				},
			},
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name: "Invalid regex pattern",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					"[invalid": {
						TotalShards:  512,
						ActiveShards: []uint64{0},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid regex",
		},
		{
			name: "Zero totalShards",
			config: &ShardingConfigV2{
				Topics: map[string]*TopicShardingConfig{
					".*": {
						TotalShards:  0,
						ActiveShards: []uint64{0},
					},
				},
			},
			expectError: true,
			errorMsg:    "totalShards must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.compilePatterns()
			if err == nil {
				err = tt.config.validate()
			}

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

func TestBatchProcessing(t *testing.T) {
	config := &ShardingConfigV2{
		Topics: map[string]*TopicShardingConfig{
			".*attestation.*": {
				TotalShards:  512,
				ActiveShards: []uint64{0, 1, 2, 3}, // 4 shards
			},
		},
	}

	err := config.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(config, true)
	require.NoError(t, err)

	// Create batch of events
	events := []ShardableEvent{
		{MsgID: "msg1", Topic: "beacon_attestation_1"},
		{MsgID: "msg2", Topic: "beacon_attestation_2"},
		{MsgID: "msg3", Topic: "unknown_topic"},
		{MsgID: "msg4", Topic: "beacon_attestation_3"},
	}

	// Process batch
	results := sharder.ShouldProcessBatch(xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE, events)

	assert.Len(t, results, len(events), "Should return result for each event")

	// At least one should be false (since we only have 4 active shards out of 512)
	hasFalse := false

	for _, result := range results {
		if !result {
			hasFalse = true

			break
		}
	}

	assert.True(t, hasFalse, "With limited shards, some events should be filtered")
}

// Helper functions

func generateShardRange(start, end uint64) []uint64 {
	result := make([]uint64, 0, end-start+1)
	for i := start; i <= end; i++ {
		result = append(result, i)
	}

	return result
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}

	return x
}

func TestTopicShardingConfigUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected *TopicShardingConfig
		wantErr  bool
	}{
		{
			name: "range syntax",
			yaml: `
totalShards: 512
activeShards: ["0-25"]
`,
			expected: &TopicShardingConfig{
				TotalShards:  512,
				ActiveShards: generateShardRange(0, 25),
			},
			wantErr: false,
		},
		{
			name: "mixed ranges and numbers",
			yaml: `
totalShards: 512
activeShards: ["0-10", "20-30", 100, 200]
`,
			expected: &TopicShardingConfig{
				TotalShards: 512,
				ActiveShards: func() []uint64 {
					shards := append(generateShardRange(0, 10), generateShardRange(20, 30)...)
					shards = append(shards, 100, 200)

					return shards
				}(),
			},
			wantErr: false,
		},
		{
			name: "single numbers",
			yaml: `
totalShards: 512
activeShards: [0, 5, 10, 15]
`,
			expected: &TopicShardingConfig{
				TotalShards:  512,
				ActiveShards: []uint64{0, 5, 10, 15},
			},
			wantErr: false,
		},
		{
			name: "invalid range format",
			yaml: `
totalShards: 512
activeShards: ["0-10-20"]
`,
			wantErr: true,
		},
		{
			name: "invalid range order",
			yaml: `
totalShards: 512
activeShards: ["10-5"]
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config TopicShardingConfig
			err := yaml.Unmarshal([]byte(tt.yaml), &config)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.TotalShards, config.TotalShards)
			assert.Equal(t, tt.expected.ActiveShards, config.ActiveShards)
		})
	}
}
