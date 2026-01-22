package clmimicry

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMsgIDBase = "test-msg-id"

func TestParseChancePercentage(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  float64
		wantError bool
	}{
		{
			name:      "2 percent",
			input:     "2%",
			expected:  0.02,
			wantError: false,
		},
		{
			name:      "0.5 percent",
			input:     "0.5%",
			expected:  0.005,
			wantError: false,
		},
		{
			name:      "100 percent",
			input:     "100%",
			expected:  1.0,
			wantError: false,
		},
		{
			name:      "0 percent",
			input:     "0%",
			expected:  0.0,
			wantError: false,
		},
		{
			name:      "50 percent",
			input:     "50%",
			expected:  0.5,
			wantError: false,
		},
		{
			name:      "2.5 percent",
			input:     "2.5%",
			expected:  0.025,
			wantError: false,
		},
		{
			name:      "with spaces",
			input:     "  10%  ",
			expected:  0.1,
			wantError: false,
		},
		{
			name:      "missing percent sign",
			input:     "2",
			expected:  0,
			wantError: true,
		},
		{
			name:      "invalid value",
			input:     "invalid%",
			expected:  0,
			wantError: true,
		},
		{
			name:      "over 100 percent",
			input:     "101%",
			expected:  0,
			wantError: true,
		},
		{
			name:      "negative percent",
			input:     "-5%",
			expected:  0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseChancePercentage(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.0001)
			}
		})
	}
}

func TestIsValidEventTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid event type",
			input:    "LIBP2P_TRACE_RPC_META_CONTROL_IWANT",
			expected: true,
		},
		{
			name:     "valid with wildcard",
			input:    "LIBP2P_TRACE_*",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "lowercase",
			input:    "libp2p_trace",
			expected: false,
		},
		{
			name:     "no underscore",
			input:    "LIBP2PTRACE",
			expected: false,
		},
		{
			name:     "regex pattern",
			input:    ".*beacon.*",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEventTypeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRandomSamplingConfigCompilePatterns(t *testing.T) {
	tests := []struct {
		name      string
		config    RandomSamplingConfig
		wantError bool
	}{
		{
			name: "valid GroupC pattern",
			config: RandomSamplingConfig{
				Patterns: map[string]*RandomSamplingPatternConfig{
					"LIBP2P_TRACE_RPC_META_CONTROL_IWANT": {Chance: "2%"},
				},
			},
			wantError: false,
		},
		{
			name: "valid GroupA/B pattern with topic",
			config: RandomSamplingConfig{
				Patterns: map[string]*RandomSamplingPatternConfig{
					"LIBP2P_TRACE_RPC_META_CONTROL_IHAVE*:.*attestation.*": {Chance: "5%"},
				},
			},
			wantError: false,
		},
		{
			name: "valid topic-only pattern",
			config: RandomSamplingConfig{
				Patterns: map[string]*RandomSamplingPatternConfig{
					".*beacon_block.*": {Chance: "10%"},
				},
			},
			wantError: false,
		},
		{
			name: "multiple patterns",
			config: RandomSamplingConfig{
				Patterns: map[string]*RandomSamplingPatternConfig{
					"LIBP2P_TRACE_RPC_META_CONTROL_IWANT":                  {Chance: "2%"},
					"LIBP2P_TRACE_RPC_META_CONTROL_IHAVE*:.*attestation.*": {Chance: "5%"},
					".*": {Chance: "0.5%"},
				},
			},
			wantError: false,
		},
		{
			name: "invalid chance format",
			config: RandomSamplingConfig{
				Patterns: map[string]*RandomSamplingPatternConfig{
					"LIBP2P_TRACE_RPC_META_CONTROL_IWANT": {Chance: "invalid"},
				},
			},
			wantError: true,
		},
		{
			name: "invalid regex pattern",
			config: RandomSamplingConfig{
				Patterns: map[string]*RandomSamplingPatternConfig{
					"LIBP2P_TRACE_*:[invalid(": {Chance: "2%"},
				},
			},
			wantError: true,
		},
		{
			name:      "nil patterns",
			config:    RandomSamplingConfig{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.compilePatterns()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRandomSamplingFindMatchingPattern(t *testing.T) {
	config := RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{
			"LIBP2P_TRACE_RPC_META_CONTROL_IWANT":                  {Chance: "2%"},
			"LIBP2P_TRACE_RPC_META_CONTROL_IHAVE*:.*attestation.*": {Chance: "5%"},
			".*beacon_block.*":                                     {Chance: "10%"},
		},
	}

	err := config.compilePatterns()
	require.NoError(t, err)

	tests := []struct {
		name           string
		topic          string
		eventType      xatu.Event_Name
		expectMatch    bool
		expectedChance float64
	}{
		{
			name:           "GroupC IWANT match - no topic",
			topic:          "",
			eventType:      xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
			expectMatch:    true,
			expectedChance: 0.02,
		},
		{
			name:           "GroupA IHAVE match with attestation topic",
			topic:          "/eth2/mainnet/beacon_attestation/1/ssz_snappy",
			eventType:      xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE,
			expectMatch:    true,
			expectedChance: 0.05,
		},
		{
			name:           "Topic pattern match for beacon block",
			topic:          "/eth2/mainnet/beacon_block/ssz_snappy",
			eventType:      xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			expectMatch:    true,
			expectedChance: 0.10,
		},
		{
			name:        "No match for unrelated event type and topic",
			topic:       "/eth2/mainnet/sync_committee/1/ssz_snappy",
			eventType:   xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION,
			expectMatch: false,
		},
		{
			name:        "GroupC IDONTWANT - no match",
			topic:       "",
			eventType:   xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT,
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.findMatchingPattern(tt.topic, tt.eventType)
			if tt.expectMatch {
				require.NotNil(t, result)
				assert.InDelta(t, tt.expectedChance, result.parsedChance, 0.0001)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestRandomSamplingNoDuplicates(t *testing.T) {
	// Test that when deterministic sharding accepts an event,
	// random sampling is NOT invoked (no duplicates)
	shardingConfig := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			".*": {
				TotalShards:  1,
				ActiveShards: []uint64{0}, // 100% deterministic sampling
			},
		},
	}

	err := shardingConfig.compilePatterns()
	require.NoError(t, err)

	randomConfig := &RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{
			".*": {Chance: "100%"}, // 100% random sampling
		},
	}

	err = randomConfig.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(shardingConfig, randomConfig, true)
	require.NoError(t, err)

	// Should be accepted by deterministic sharding
	// The reason should NOT contain "random"
	shouldProcess, reason := sharder.ShouldProcess(
		xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
		"test-msg-id",
		"/eth2/mainnet/beacon_block/ssz_snappy",
	)

	assert.True(t, shouldProcess)
	assert.NotContains(t, reason, "random")
	assert.Contains(t, reason, "group_a")
}

func TestRandomSamplingSecondChance(t *testing.T) {
	// Test that when deterministic sharding rejects an event,
	// random sampling is tried
	//
	// We configure deterministic sharding to only accept shard 511,
	// and use a message ID that we know hashes to a different shard.
	shardingConfig := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			".*": {
				TotalShards:  512,
				ActiveShards: []uint64{511}, // Only shard 511 is active
			},
		},
	}

	err := shardingConfig.compilePatterns()
	require.NoError(t, err)

	randomConfig := &RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{
			".*": {Chance: "100%"}, // 100% random sampling
		},
	}

	err = randomConfig.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(shardingConfig, randomConfig, true)
	require.NoError(t, err)

	// Find a message ID that doesn't hash to shard 511
	testMsgID := testMsgIDBase
	for GetShard(testMsgID, 512) == 511 {
		testMsgID = testMsgID + "x"
	}

	// Deterministic should reject (msgID doesn't hash to shard 511)
	// Random sampling at 100% should accept
	shouldProcess, reason := sharder.ShouldProcess(
		xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
		testMsgID,
		"/eth2/mainnet/beacon_block/ssz_snappy",
	)

	assert.True(t, shouldProcess)
	assert.Contains(t, reason, "random_sampled")
}

func TestRandomSamplingGroupC(t *testing.T) {
	// Test GroupC events (IWANT) work with random sampling
	shardingConfig := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			"LIBP2P_TRACE_RPC_META_CONTROL_IWANT": {
				TotalShards:  512,
				ActiveShards: []uint64{511}, // Only shard 511 is active
			},
		},
	}

	err := shardingConfig.compilePatterns()
	require.NoError(t, err)

	randomConfig := &RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{
			"LIBP2P_TRACE_RPC_META_CONTROL_IWANT": {Chance: "100%"},
		},
	}

	err = randomConfig.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(shardingConfig, randomConfig, true)
	require.NoError(t, err)

	// Find a message ID that doesn't hash to shard 511
	testMsgID := testMsgIDBase
	for GetShard(testMsgID, 512) == 511 {
		testMsgID = testMsgID + "x"
	}

	// GroupC event with no topic - should be accepted by random sampling
	shouldProcess, reason := sharder.ShouldProcess(
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
		testMsgID,
		"", // GroupC events have no topic
	)

	assert.True(t, shouldProcess)
	assert.Contains(t, reason, "random_sampled")
}

func TestRandomSamplingDisabled(t *testing.T) {
	// Test that when random sampling config is nil, it doesn't affect deterministic
	shardingConfig := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			".*": {
				TotalShards:  1,
				ActiveShards: []uint64{0}, // 100% deterministic
			},
		},
	}

	err := shardingConfig.compilePatterns()
	require.NoError(t, err)

	// No random sampling config
	sharder, err := NewUnifiedSharder(shardingConfig, nil, true)
	require.NoError(t, err)

	assert.False(t, sharder.randomSamplingEnabled)

	shouldProcess, reason := sharder.ShouldProcess(
		xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
		"test-msg-id",
		"/eth2/mainnet/beacon_block/ssz_snappy",
	)

	assert.True(t, shouldProcess)
	assert.NotContains(t, reason, "random")
}

func TestRandomSamplingDistribution(t *testing.T) {
	// Statistical test to verify approximate random sampling rate
	shardingConfig := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			".*": {
				TotalShards:  512,
				ActiveShards: []uint64{511}, // Only shard 511 is active
			},
		},
	}

	err := shardingConfig.compilePatterns()
	require.NoError(t, err)

	randomConfig := &RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{
			".*": {Chance: "10%"}, // 10% random sampling
		},
	}

	err = randomConfig.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(shardingConfig, randomConfig, true)
	require.NoError(t, err)

	// Find a message ID that doesn't hash to shard 511
	testMsgID := testMsgIDBase
	for GetShard(testMsgID, 512) == 511 {
		testMsgID = testMsgID + "x"
	}

	// Run many trials
	const trials = 10000

	accepted := 0

	for i := 0; i < trials; i++ {
		shouldProcess, _ := sharder.ShouldProcess(
			xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			testMsgID, // Same msgID that deterministic always rejects
			"/eth2/mainnet/beacon_block/ssz_snappy",
		)
		if shouldProcess {
			accepted++
		}
	}

	// With 10% sampling, we expect ~1000 accepted out of 10000
	// Allow for statistical variance (8% to 12%)
	acceptedRate := float64(accepted) / float64(trials)
	assert.InDelta(t, 0.10, acceptedRate, 0.02, "Expected ~10%% acceptance rate, got %.2f%%", acceptedRate*100)
}

func TestRandomSamplingWildcardEventType(t *testing.T) {
	// Test that wildcard event type patterns match multiple event types
	config := RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{
			"LIBP2P_TRACE_RPC_META_*:.*attestation.*": {Chance: "100%"},
		},
	}

	err := config.compilePatterns()
	require.NoError(t, err)

	topic := "/eth2/mainnet/beacon_attestation/1/ssz_snappy"

	// Should match IHAVE
	result := config.findMatchingPattern(topic, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE)
	require.NotNil(t, result, "IHAVE should match wildcard pattern")

	// Should match IWANT (even though it's GroupC, the wildcard should still match the event type)
	result = config.findMatchingPattern(topic, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT)
	require.NotNil(t, result, "IWANT should match wildcard pattern")

	// Should NOT match GOSSIPSUB events
	result = config.findMatchingPattern(topic, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION)
	assert.Nil(t, result, "GOSSIPSUB should not match RPC_META wildcard")
}

func TestRandomSamplingZeroPercent(t *testing.T) {
	// Test that 0% chance never accepts
	shardingConfig := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			".*": {
				TotalShards:  512,
				ActiveShards: []uint64{511}, // Only shard 511 is active
			},
		},
	}

	err := shardingConfig.compilePatterns()
	require.NoError(t, err)

	randomConfig := &RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{
			".*": {Chance: "0%"}, // 0% should never accept
		},
	}

	err = randomConfig.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(shardingConfig, randomConfig, true)
	require.NoError(t, err)

	// Find a message ID that doesn't hash to shard 511
	testMsgID := testMsgIDBase
	for GetShard(testMsgID, 512) == 511 {
		testMsgID = testMsgID + "x"
	}

	// Run many trials - none should be accepted
	for i := 0; i < 1000; i++ {
		shouldProcess, _ := sharder.ShouldProcess(
			xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			testMsgID,
			"/eth2/mainnet/beacon_block/ssz_snappy",
		)
		assert.False(t, shouldProcess, "0%% chance should never accept")
	}
}

func TestRandomSamplingGroupB(t *testing.T) {
	// Test that topic-only patterns work for random sampling
	shardingConfig := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			".*": {
				TotalShards:  512,
				ActiveShards: []uint64{511}, // Only shard 511 is active
			},
		},
	}

	err := shardingConfig.compilePatterns()
	require.NoError(t, err)

	randomConfig := &RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{
			".*beacon_block.*": {Chance: "100%"}, // Topic-only pattern
		},
	}

	err = randomConfig.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(shardingConfig, randomConfig, true)
	require.NoError(t, err)

	// Find a message ID that doesn't hash to shard 511
	testMsgID := testMsgIDBase
	for GetShard(testMsgID, 512) == 511 {
		testMsgID = testMsgID + "x"
	}

	// Topic-only pattern should match based on topic
	shouldProcess, reason := sharder.ShouldProcess(
		xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
		testMsgID,
		"/eth2/mainnet/beacon_block/ssz_snappy",
	)

	assert.True(t, shouldProcess)
	assert.Contains(t, reason, "random_sampled")
}

func TestRandomSamplingEmptyPatterns(t *testing.T) {
	// Test that empty patterns map (not nil) disables random sampling
	randomConfig := &RandomSamplingConfig{
		Patterns: map[string]*RandomSamplingPatternConfig{}, // Empty, not nil
	}

	err := randomConfig.compilePatterns()
	require.NoError(t, err)

	shardingConfig := &ShardingConfig{
		Topics: map[string]*TopicShardingConfig{
			".*": {
				TotalShards:  512,
				ActiveShards: []uint64{511},
			},
		},
	}

	err = shardingConfig.compilePatterns()
	require.NoError(t, err)

	sharder, err := NewUnifiedSharder(shardingConfig, randomConfig, true)
	require.NoError(t, err)

	assert.False(t, sharder.randomSamplingEnabled, "Empty patterns should disable random sampling")
}

func TestRandomSamplingLogSummary(t *testing.T) {
	tests := []struct {
		name     string
		config   RandomSamplingConfig
		contains []string
	}{
		{
			name:     "empty config",
			config:   RandomSamplingConfig{},
			contains: []string{"disabled"},
		},
		{
			name: "with patterns",
			config: RandomSamplingConfig{
				Patterns: map[string]*RandomSamplingPatternConfig{
					"LIBP2P_TRACE_RPC_META_CONTROL_IWANT": {Chance: "2%"},
				},
			},
			contains: []string{"enabled", "1 patterns", "2%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.config.LogSummary()
			for _, substr := range tt.contains {
				assert.Contains(t, summary, substr)
			}
		})
	}
}
