package clmimicry

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/probe-lab/hermes/host"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPayload implements the necessary methods to work with getMsgID.
type MockPayload struct {
	MsgID string
}

func TestShouldTraceMessage(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	// Find message IDs that map to specific shards for our tests
	msgIDForShard2 := findMsgIDForShard(2, 64)
	msgIDForShard4 := findMsgIDForShard(4, 64)
	uint64Ptr := uint64(64)
	uint64Ptr4 := uint64(4)

	tests := []struct {
		name      string
		mimicry   *Mimicry
		msgID     string
		eventType string
		networkID uint64
		expected  bool
	}{
		{
			name: "empty msgID always returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:     &uint64Ptr,
								ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
								ActiveShards:    []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test1"),
			},
			msgID:     "",
			eventType: "test_event",
			networkID: networks.DeriveFromID(1).ID, // mainnet
			expected:  true,
		},
		{
			name: "traces disabled returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: false,
					},
				},
				metrics: NewMetrics("test2"),
			},
			msgID:     "0x1234",
			eventType: "test_event",
			networkID: networks.DeriveFromID(1).ID, // mainnet
			expected:  true,
		},
		{
			name: "no matching topic returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"other_event": {
								TotalShards:     &uint64Ptr,
								ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
								ActiveShards:    []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test3"),
			},
			msgID:     "0x1234",
			eventType: "test_event",
			networkID: networks.DeriveFromID(1).ID, // mainnet
			expected:  true,
		},
		{
			name: "shard in active shards returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:     &uint64Ptr,
								ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
								ActiveShards:    []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test4"),
			},
			msgID:     msgIDForShard2,
			eventType: "test_event",
			networkID: networks.DeriveFromID(1).ID, // mainnet
			expected:  true,
		},
		{
			name: "shard not in active shards returns false",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:     &uint64Ptr,
								ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
								ActiveShards:    []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test5"),
			},
			msgID:     msgIDForShard4,
			eventType: "test_event",
			networkID: networks.DeriveFromID(1).ID, // mainnet
			expected:  false,
		},
		{
			name: "empty network uses unknown",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:     &uint64Ptr,
								ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
								ActiveShards:    []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test6"),
			},
			msgID:     msgIDForShard2,
			eventType: "test_event",
			networkID: 0, // unknown
			expected:  true,
		},
		{
			name: "regex pattern matching",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_.*": {
								TotalShards:     &uint64Ptr,
								ActiveShardsRaw: &ActiveShardsConfig{1, 2, 3},
								ActiveShards:    []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test7"),
			},
			msgID:     msgIDForShard2,
			eventType: "test_something",
			networkID: networks.DeriveFromID(1).ID, // mainnet
			expected:  true,
		},
		{
			name: "all shards active skips hashing and returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:     &uint64Ptr4,
								ActiveShardsRaw: &ActiveShardsConfig{0, 1, 2, 3},
								// All shards 0-3 are active
								ActiveShards: []uint64{0, 1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test8"),
			},
			// This would normally map to a shard that might or might not be active,
			// but since all shards are active, it should skip the hash calculation
			// and return true regardless of the actual shard
			msgID:     "0xdoesntmatter",
			eventType: "test_event",
			networkID: networks.DeriveFromID(1).ID, // mainnet
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock objects for the new parameter structure
			event := createMockTraceEvent(tt.msgID)
			clientMeta := createMockClientMeta(tt.networkID)

			result := tt.mimicry.ShouldTraceMessage(event, clientMeta, tt.eventType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSkipsSipHashIfAllShardsActive specifically tests that we skip hashing when all shards are active.
func TestSkipsSipHashIfAllShardsActive(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	// Small total shards for easier testing.
	totalShards := uint64(8)

	// Create a list of all shards from 0 to totalShards-1.
	allShards := make([]uint64, totalShards)
	for i := uint64(0); i < totalShards; i++ {
		allShards[i] = i
	}

	m := &Mimicry{
		Config: &Config{
			Traces: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"test_event": {
						TotalShards:  &totalShards,
						ActiveShards: allShards,
					},
				},
			},
		},
		metrics: NewMetrics("test_all_active_shards1"),
	}

	// Even a message ID that would hash to an inactive shard should return true
	// because the optimization should skip the hashing entirely.
	mockEvent := createMockTraceEvent("0xanymessage")
	mockClientMeta := createMockClientMeta(networks.DeriveFromID(1).ID) // mainnet
	result := m.ShouldTraceMessage(mockEvent, mockClientMeta, "test_event")
	assert.True(t, result, "When all shards are active, any message should be processed")

	// Now let's remove one shard and verify the optimization doesn't apply.
	partialShards := make([]uint64, totalShards-1)
	copy(partialShards, allShards[:totalShards-1])

	// Create a new config with one shard missing.
	m = &Mimicry{
		Config: &Config{
			Traces: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"test_event": {
						TotalShards:  &totalShards,
						ActiveShards: partialShards,
					},
				},
			},
		},
		metrics: NewMetrics("test_all_active_shards2"),
	}

	// Find a message that would hash to the inactive shard.
	inactiveShard := totalShards - 1
	msgIDForInactiveShard := findMsgIDForShard(inactiveShard, totalShards)

	mockEvent = createMockTraceEvent(msgIDForInactiveShard)
	result = m.ShouldTraceMessage(mockEvent, mockClientMeta, "test_event")
	assert.False(t, result, "When not all shards are active, messages for inactive shards should be skipped")
}

// Note: TestShouldTraceMessageWithDifferentShardingKeys removed - covered by TestComprehensiveEventSharding in trace_integration_test.go

// TestSimpleConfigBackwardCompatibility tests that old simple configs still work
func TestSimpleConfigBackwardCompatibility(t *testing.T) {
	totalShards := uint64(64)
	activeShards := ActiveShardsConfig{0, 1, 2}

	config := TracesConfig{
		Enabled: true,
		Topics: map[string]TopicConfig{
			"beacon_block": {
				TotalShards:     &totalShards,
				ActiveShardsRaw: &activeShards,
				ShardingKey:     "MsgID",
			},
		},
	}

	// Should validate without error
	err := config.Validate()
	assert.NoError(t, err)

	// Should compile without error
	err = config.CompilePatterns()
	assert.NoError(t, err)

	// Should find matching config
	matchedConfig, found := config.FindMatchingTopicConfig("beacon_block")
	assert.True(t, found)
	assert.NotNil(t, matchedConfig)

	// Should produce expected log summary
	summary := config.LogSummary()
	assert.Contains(t, summary, "Pattern 'beacon_block': 3/64 shards active")
	assert.NotContains(t, summary, "HIERARCHICAL")
}

// findMsgIDForShard finds a message ID that maps to a specific shard.
func findMsgIDForShard(shard, totalShards uint64) string {
	// Start with a base message ID and increment until we find one that maps to the desired shard.
	for i := 0; i < 1000; i++ {
		msgID := fmt.Sprintf("0xtest%d", i)
		if GetShard(msgID, totalShards) == shard {
			return msgID
		}
	}

	return fmt.Sprintf("0xshard%d", shard) // Fallback, though this might not map to the desired shard.
}

// createMockTraceEvent creates a mock TraceEvent with the given msgID
func createMockTraceEvent(msgID string) *host.TraceEvent {
	return &host.TraceEvent{
		Payload: &MockPayload{
			MsgID: msgID,
		},
	}
}

// createMockClientMeta creates a mock ClientMeta with the given network ID
func createMockClientMeta(networkID uint64) *xatu.ClientMeta {
	return &xatu.ClientMeta{
		Name:           "test-client",
		Id:             "test-id",
		Implementation: "test-impl",
		Os:             "test-os",
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Id: networkID,
			},
		},
	}
}

// TestTraceDisabledScenario tests behavior when traces are disabled
func TestTraceDisabledScenario(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	config := &Config{
		Traces: TracesConfig{
			Enabled: false, // Traces disabled
			Topics: map[string]TopicConfig{
				"test_pattern": {
					Topics: &TopicsConfig{
						GossipTopics: map[string]GossipTopicConfig{
							".*test.*": {
								TotalShards:     4,
								ActiveShardsRaw: ActiveShardsConfig{0, 1},
								ShardingKey:     "MsgID",
							},
						},
					},
				},
			},
		},
	}

	err := config.Traces.Validate()
	require.NoError(t, err)
	err = config.Traces.CompilePatterns()
	require.NoError(t, err)

	mimicry := &Mimicry{
		Config:  config,
		metrics: NewMetrics("test_disabled"),
	}

	// Test findCompiledTopicConfig with traces disabled
	result := mimicry.findCompiledTopicConfig(xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String())
	assert.Nil(t, result, "Should return nil when traces are disabled")

	// Verify that ShouldTraceMessage still works (returns true when disabled)
	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	event := &host.TraceEvent{
		Type:      "DUPLICATE_MESSAGE",
		PeerID:    testPeerID,
		Timestamp: time.Now(),
		Payload: map[string]any{
			"MsgID": "test_msg",
			"Topic": "/eth2/test/topic",
		},
	}
	clientMeta := createMockClientMeta(1)

	shouldTrace := mimicry.ShouldTraceMessage(event, clientMeta, xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String())
	assert.True(t, shouldTrace, "Should return true when traces are disabled")
}

// TestNilTotalShardsEdgeCase tests simple config behavior with nil TotalShards
func TestNilTotalShardsEdgeCase(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	// Create a config that passes validation, then manually set TotalShards to nil
	// to test the runtime nil check behavior
	totalShards := uint64(4)
	config := &Config{
		Traces: TracesConfig{
			Enabled: true,
			Topics: map[string]TopicConfig{
				"(?i).*recv_rpc.*": {
					TotalShards:     &totalShards, // Valid for validation
					ActiveShardsRaw: &ActiveShardsConfig{0, 1},
					ShardingKey:     "MsgID",
				},
			},
		},
	}

	err := config.Traces.Validate()
	require.NoError(t, err)
	err = config.Traces.CompilePatterns()
	require.NoError(t, err)

	// Now manually set TotalShards to nil to test the runtime behavior
	for pattern := range config.Traces.Topics {
		topicConfig := config.Traces.Topics[pattern]
		topicConfig.TotalShards = nil // This should trigger the nil check at runtime
		config.Traces.Topics[pattern] = topicConfig
	}

	mimicry := &Mimicry{
		Config:  config,
		metrics: NewMetrics("test_nil_shards"),
	}

	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	event := &host.TraceEvent{
		Type:      "RECV_RPC",
		PeerID:    testPeerID,
		Timestamp: time.Now(),
		Payload:   map[string]any{},
	}
	clientMeta := createMockClientMeta(1)

	t.Run("nil_totalShards_shouldTraceWithSimpleConfig", func(t *testing.T) {
		// Test the shouldTraceWithSimpleConfig path directly
		topicConfig, found := config.Traces.FindMatchingTopicConfig(xatu.Event_LIBP2P_TRACE_RECV_RPC.String())
		require.True(t, found)

		result := mimicry.shouldTraceWithSimpleConfig(event, clientMeta, topicConfig, xatu.Event_LIBP2P_TRACE_RECV_RPC.String())
		assert.True(t, result, "Should return true when TotalShards is nil in simple config")
	})
}
