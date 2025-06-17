package clmimicry

import (
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/probe-lab/hermes/host"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestHierarchicalShardingIntegration tests the complete hierarchical sharding flow
func TestHierarchicalShardingIntegration(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	// Create test data
	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	testTime := time.Now()

	// Different message IDs that will hash to different shards
	beaconBlockMsgID := "beacon_block_msg_123"
	beaconAttestationMsgID := "beacon_attestation_msg_456"
	otherMsgID := "other_msg_789"

	tests := []struct {
		name        string
		gossipTopic string
		msgID       string
		expected    bool
		description string
	}{
		{
			name:        "beacon_block topic - 100% sampling",
			gossipTopic: "/eth2/4a26c58b/beacon_block/ssz_snappy",
			msgID:       beaconBlockMsgID,
			expected:    true,
			description: "Beacon block topics should have 100% sampling (all 1 shard active)",
		},
		{
			name:        "beacon_attestation topic - 50% sampling",
			gossipTopic: "/eth2/4a26c58b/beacon_attestation_1/ssz_snappy",
			msgID:       beaconAttestationMsgID,
			expected:    true, // This might be true or false depending on which shard the message hashes to
			description: "Beacon attestation topics should have 50% sampling (2 of 4 shards active)",
		},
		{
			name:        "other topic - fallback 1/512 sampling",
			gossipTopic: "/eth2/4a26c58b/voluntary_exit/ssz_snappy",
			msgID:       otherMsgID,
			expected:    false, // Very likely to be false with 1/512 sampling
			description: "Other topics should use fallback configuration with 1/512 sampling",
		},
		{
			name:        "no topic - fallback 1/512 sampling",
			gossipTopic: "",
			msgID:       otherMsgID,
			expected:    false, // Very likely to be false with 1/512 sampling
			description: "Events without gossip topic should use fallback configuration",
		},
	}

	// Create hierarchical configuration
	config := &Config{
		Traces: TracesConfig{
			Enabled: true,
			Topics: map[string]TopicConfig{
				"(?i).*duplicate_message.*": {
					Topics: &TopicsConfig{
						GossipTopics: map[string]GossipTopicConfig{
							".*beacon_block.*": {
								TotalShards:     1,
								ActiveShardsRaw: ActiveShardsConfig{0},
								ShardingKey:     "MsgID",
							},
							".*beacon_attestation.*": {
								TotalShards:     4,
								ActiveShardsRaw: ActiveShardsConfig{0, 1}, // 50% sampling
								ShardingKey:     "MsgID",
							},
						},
						Fallback: &GossipTopicConfig{
							TotalShards:     512,
							ActiveShardsRaw: ActiveShardsConfig{0}, // 1/512 sampling
							ShardingKey:     "MsgID",
						},
					},
				},
			},
		},
	}

	// Validate and compile the configuration
	err := config.Traces.Validate()
	require.NoError(t, err)
	err = config.Traces.CompilePatterns()
	require.NoError(t, err)

	// Create mimicry instance
	mimicry := &Mimicry{
		Config:  config,
		metrics: NewMetrics("test_hierarchical"),
	}

	clientMeta := createMockClientMeta(1) // mainnet

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create event with map payload (simulating hermes events)
			event := &host.TraceEvent{
				Type:      "DUPLICATE_MESSAGE",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: map[string]any{
					"MsgID": tt.msgID,
					"Topic": tt.gossipTopic,
				},
			}

			result := mimicry.ShouldTraceMessage(
				event,
				clientMeta,
				xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String(),
			)

			// For deterministic tests, we can verify specific cases
			if tt.gossipTopic != "" && strings.Contains(tt.gossipTopic, "beacon_block") {
				// Beacon block should always be sampled (100% sampling)
				assert.True(t, result, "Beacon block messages should always be sampled")
			} else if tt.gossipTopic != "" && strings.Contains(tt.gossipTopic, "beacon_attestation") {
				// Beacon attestation has 50% sampling (2 out of 4 shards)
				expectedShard := GetShard(tt.msgID, 4)
				expectedResult := expectedShard <= 1 // shards 0 and 1 are active
				assert.Equal(t, expectedResult, result, "Beacon attestation sampling should match shard calculation")
			} else {
				// For fallback config with 1/512 sampling, we can test specific message IDs
				expectedShard := GetShard(tt.msgID, 512)
				expectedResult := expectedShard == 0
				assert.Equal(t, expectedResult, result, "Fallback sampling should match shard calculation")
			}

			t.Logf("Test: %s, Topic: %s, MsgID: %s, Result: %v", tt.name, tt.gossipTopic, tt.msgID, result)
		})
	}
}

// TestHierarchicalShardingWithProtobufPayload tests hierarchical sharding with protobuf-style payloads
func TestHierarchicalShardingWithProtobufPayload(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	testTime := time.Now()
	msgID := "protobuf_test_msg"

	// Create hierarchical configuration with specific shard for deterministic testing
	config := &Config{
		Traces: TracesConfig{
			Enabled: true,
			Topics: map[string]TopicConfig{
				"(?i).*deliver_message.*": {
					Topics: &TopicsConfig{
						GossipTopics: map[string]GossipTopicConfig{
							".*blob_sidecar.*": {
								TotalShards:     1,
								ActiveShardsRaw: ActiveShardsConfig{0},
								ShardingKey:     "MsgID",
							},
						},
						Fallback: &GossipTopicConfig{
							TotalShards:     2,
							ActiveShardsRaw: ActiveShardsConfig{0}, // 50% sampling
							ShardingKey:     "MsgID",
						},
					},
				},
			},
		},
	}

	// Validate and compile the configuration
	err := config.Traces.Validate()
	require.NoError(t, err)
	err = config.Traces.CompilePatterns()
	require.NoError(t, err)

	mimicry := &Mimicry{
		Config:  config,
		metrics: NewMetrics("test_protobuf"),
	}

	clientMeta := createMockClientMeta(1) // mainnet

	tests := []struct {
		name        string
		payload     interface{}
		expected    bool
		description string
	}{
		{
			name: "protobuf payload with blob_sidecar topic",
			payload: &MockPayloadWithTopic{
				Topic: &MockStringWrapper{Value: "/eth2/4a26c58b/blob_sidecar_0/ssz_snappy"},
			},
			expected:    true,
			description: "Blob sidecar should have 100% sampling",
		},
		{
			name: "protobuf payload with other topic",
			payload: &MockPayloadWithTopic{
				Topic: &MockStringWrapper{Value: "/eth2/4a26c58b/sync_committee_contribution_and_proof/ssz_snappy"},
			},
			expected:    GetShard(msgID, 2) == 0, // Use fallback config
			description: "Other topics should use fallback configuration",
		},
		{
			name: "protobuf payload without topic",
			payload: &MockPayloadWithTopic{
				Topic: nil,
			},
			expected:    GetShard(msgID, 2) == 0, // Use fallback config
			description: "Payloads without topic should use fallback configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &host.TraceEvent{
				Type:      "DELIVER_MESSAGE",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload:   tt.payload,
			}

			// Add MsgID to map payload for GetShardingKey to work
			if mapPayload, ok := tt.payload.(map[string]any); ok {
				mapPayload["MsgID"] = msgID
			}

			result := mimicry.ShouldTraceMessage(
				event,
				clientMeta,
				xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE.String(),
			)

			assert.Equal(t, tt.expected, result, tt.description)
			t.Logf("Test: %s, Expected: %v, Result: %v", tt.name, tt.expected, result)
		})
	}
}

// TestHierarchicalRPCMetaFiltering tests hierarchical filtering of RPC meta messages
func TestHierarchicalRPCMetaFiltering(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	testTime := time.Now()

	// Create hierarchical configuration
	config := &Config{
		Traces: TracesConfig{
			Enabled: true,
			Topics: map[string]TopicConfig{
				"(?i).*recv_rpc.*": {
					Topics: &TopicsConfig{
						Fallback: &GossipTopicConfig{
							TotalShards:     4,
							ActiveShardsRaw: ActiveShardsConfig{0, 1}, // 50% sampling
							ShardingKey:     "MsgID",
						},
					},
				},
			},
		},
	}

	// Validate and compile the configuration
	err := config.Traces.Validate()
	require.NoError(t, err)
	err = config.Traces.CompilePatterns()
	require.NoError(t, err)

	mimicry := &Mimicry{
		Config:  config,
		metrics: NewMetrics("test_rpc_meta"),
	}

	clientMeta := createMockClientMeta(1) // mainnet

	// Create test message IDs
	messageIDs := []*wrapperspb.StringValue{
		{Value: "msg_1"},
		{Value: "msg_2"},
		{Value: "msg_3"},
		{Value: "msg_4"},
		{Value: "msg_5"},
		{Value: "msg_6"},
	}

	event := &host.TraceEvent{
		Type:      "RECV_RPC",
		PeerID:    testPeerID,
		Timestamp: testTime,
		Payload:   map[string]any{},
	}

	result, err := mimicry.ShouldTraceRPCMetaMessages(
		event,
		clientMeta,
		xatu.Event_LIBP2P_TRACE_RECV_RPC.String(),
		messageIDs,
	)

	require.NoError(t, err)

	// Check that the result contains the expected messages based on sharding
	expectedCount := 0
	for _, msgID := range messageIDs {
		shard := GetShard(msgID.Value, 4)
		if shard == 0 || shard == 1 {
			expectedCount++
		}
	}

	assert.Equal(t, expectedCount, len(result), "Should filter messages based on hierarchical configuration")

	// Verify that the returned messages are the correct ones
	for _, filtered := range result {
		shard := GetShard(filtered.MessageID.Value, 4)
		assert.True(t, shard == 0 || shard == 1, "Filtered message should be in active shards")
	}

	t.Logf("Filtered %d out of %d messages", len(result), len(messageIDs))
}

// TestHierarchicalRPCMetaMessageFiltering tests hierarchical filtering of RPC meta messages with topics
func TestHierarchicalRPCMetaMessageFiltering(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	testTime := time.Now()

	// Create hierarchical configuration
	config := &Config{
		Traces: TracesConfig{
			Enabled: true,
			Topics: map[string]TopicConfig{
				"(?i).*recv_rpc.*": {
					Topics: &TopicsConfig{
						GossipTopics: map[string]GossipTopicConfig{
							".*beacon_block.*": {
								TotalShards:     1,
								ActiveShardsRaw: ActiveShardsConfig{0}, // 100% sampling
								ShardingKey:     "MsgID",
							},
							".*beacon_attestation.*": {
								TotalShards:     4,
								ActiveShardsRaw: ActiveShardsConfig{0, 1}, // 50% sampling
								ShardingKey:     "MsgID",
							},
						},
						Fallback: &GossipTopicConfig{
							TotalShards:     512,
							ActiveShardsRaw: ActiveShardsConfig{0}, // 1/512 sampling
							ShardingKey:     "MsgID",
						},
					},
				},
			},
		},
	}

	// Validate and compile the configuration
	err := config.Traces.Validate()
	require.NoError(t, err)
	err = config.Traces.CompilePatterns()
	require.NoError(t, err)

	mimicry := &Mimicry{
		Config:  config,
		metrics: NewMetrics("test_rpc_meta_hierarchical"),
	}

	clientMeta := createMockClientMeta(1) // mainnet

	// Create test messages with different topics
	messages := []RPCMetaMessageInfo{
		{
			MessageID: wrapperspb.String("beacon_block_msg_1"),
			Topic:     wrapperspb.String("/eth2/4a26c58b/beacon_block/ssz_snappy"),
		},
		{
			MessageID: wrapperspb.String("beacon_attestation_msg_2"),
			Topic:     wrapperspb.String("/eth2/4a26c58b/beacon_attestation_1/ssz_snappy"),
		},
		{
			MessageID: wrapperspb.String("other_msg_3"),
			Topic:     wrapperspb.String("/eth2/4a26c58b/voluntary_exit/ssz_snappy"),
		},
		{
			MessageID: wrapperspb.String("no_topic_msg_4"),
			Topic:     nil, // No topic - should use fallback
		},
	}

	event := &host.TraceEvent{
		Type:      "RECV_RPC",
		PeerID:    testPeerID,
		Timestamp: testTime,
		Payload:   map[string]any{},
	}

	result, err := mimicry.ShouldTraceRPCMetaMessages(
		event,
		clientMeta,
		xatu.Event_LIBP2P_TRACE_RECV_RPC.String(),
		messages,
	)

	require.NoError(t, err)

	// Verify results based on hierarchical configuration
	expectedResults := make(map[string]bool)
	for i, msg := range messages {
		msgID := msg.MessageID.Value
		var gossipTopic string
		if msg.Topic != nil {
			gossipTopic = msg.Topic.Value
		}

		var expectedSampled bool
		if strings.Contains(gossipTopic, "beacon_block") {
			// beacon_block: 100% sampling (1 shard, all active)
			expectedSampled = true
		} else if strings.Contains(gossipTopic, "beacon_attestation") {
			// beacon_attestation: 50% sampling (4 shards, 2 active: 0,1)
			shard := GetShard(msgID, 4)
			expectedSampled = shard <= 1
		} else {
			// Other topics or no topic: fallback 1/512 sampling
			shard := GetShard(msgID, 512)
			expectedSampled = shard == 0
		}

		expectedResults[msgID] = expectedSampled
		t.Logf("Message %d: ID=%s, Topic=%s, Expected=%v", i, msgID, gossipTopic, expectedSampled)
	}

	// Check that results match expectations
	resultMap := make(map[string]bool)
	for _, filtered := range result {
		resultMap[filtered.MessageID.Value] = true
	}

	for _, msg := range messages {
		msgID := msg.MessageID.Value
		expected := expectedResults[msgID]
		actual := resultMap[msgID]
		assert.Equal(t, expected, actual, "Message %s sampling result should match expectation", msgID)
	}

	t.Logf("Processed %d messages, filtered to %d results", len(messages), len(result))
}

// TestRefactoredDelegationPattern tests that ShouldTraceMessage correctly delegates to hierarchical and simple methods
func TestRefactoredDelegationPattern(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	testTime := time.Now()
	uint64Ptr := uint64(64)

	// Test simple configuration delegation
	t.Run("simple_config_delegation", func(t *testing.T) {
		config := &Config{
			Traces: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						TotalShards:     &uint64Ptr,
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

		mimicry := &Mimicry{
			Config:  config,
			metrics: NewMetrics("test_simple_delegation"),
		}

		event := &host.TraceEvent{
			Type:      "DUPLICATE_MESSAGE",
			PeerID:    testPeerID,
			Timestamp: testTime,
			Payload:   map[string]any{"MsgID": "simple_test_msg"},
		}

		clientMeta := createMockClientMeta(1)
		result := mimicry.ShouldTraceMessage(event, clientMeta, xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String())

		// Should use simple config logic
		expectedShard := GetShard("simple_test_msg", 64)
		expectedResult := expectedShard <= 1
		assert.Equal(t, expectedResult, result, "Simple config should delegate correctly")
	})

	// Test hierarchical configuration delegation
	t.Run("hierarchical_config_delegation", func(t *testing.T) {
		config := &Config{
			Traces: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						Topics: &TopicsConfig{
							GossipTopics: map[string]GossipTopicConfig{
								".*beacon_block.*": {
									TotalShards:     1,
									ActiveShardsRaw: ActiveShardsConfig{0},
									ShardingKey:     "MsgID",
								},
							},
							Fallback: &GossipTopicConfig{
								TotalShards:     512,
								ActiveShardsRaw: ActiveShardsConfig{0},
								ShardingKey:     "MsgID",
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
			metrics: NewMetrics("test_hierarchical_delegation"),
		}

		event := &host.TraceEvent{
			Type:      "DUPLICATE_MESSAGE",
			PeerID:    testPeerID,
			Timestamp: testTime,
			Payload: map[string]any{
				"MsgID": "hierarchical_test_msg",
				"Topic": "/eth2/4a26c58b/beacon_block/ssz_snappy",
			},
		}

		clientMeta := createMockClientMeta(1)
		result := mimicry.ShouldTraceMessage(event, clientMeta, xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String())

		// Should use hierarchical config logic - beacon_block should get 100% sampling
		assert.True(t, result, "Hierarchical config should delegate correctly and sample beacon_block at 100%")
	})
}

// TestConsolidatedRPCMethodInterface tests the unified ShouldTraceRPCMetaMessages interface
func TestConsolidatedRPCMethodInterface(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	testTime := time.Now()

	// Create simple configuration for testing
	uint64Ptr := uint64(4)
	config := &Config{
		Traces: TracesConfig{
			Enabled: true,
			Topics: map[string]TopicConfig{
				"(?i).*recv_rpc.*": {
					TotalShards:     &uint64Ptr,
					ActiveShardsRaw: &ActiveShardsConfig{0, 1}, // 50% sampling
					ShardingKey:     "MsgID",
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
		metrics: NewMetrics("test_consolidated"),
	}

	clientMeta := createMockClientMeta(1)
	event := &host.TraceEvent{
		Type:      "RECV_RPC",
		PeerID:    testPeerID,
		Timestamp: testTime,
		Payload:   map[string]any{},
	}

	// Test with RPCMetaMessageInfo (new format)
	t.Run("unified_interface_with_topics", func(t *testing.T) {
		messages := []RPCMetaMessageInfo{
			{MessageID: wrapperspb.String("msg_1"), Topic: wrapperspb.String("/eth2/beacon_block")},
			{MessageID: wrapperspb.String("msg_2"), Topic: wrapperspb.String("/eth2/attestation")},
		}

		result, err := mimicry.ShouldTraceRPCMetaMessages(event, clientMeta, xatu.Event_LIBP2P_TRACE_RECV_RPC.String(), messages)
		require.NoError(t, err)

		// Verify that we get results (exact count depends on sharding)
		assert.LessOrEqual(t, len(result), len(messages), "Should not get more results than input messages")
		t.Logf("Topic-aware format: filtered %d out of %d messages", len(result), len(messages))
	})

	// Test with []*wrapperspb.StringValue (message IDs only)
	t.Run("unified_interface_message_ids", func(t *testing.T) {
		messageIDs := []*wrapperspb.StringValue{
			wrapperspb.String("msg_1"),
			wrapperspb.String("msg_2"),
		}

		result, err := mimicry.ShouldTraceRPCMetaMessages(event, clientMeta, xatu.Event_LIBP2P_TRACE_RECV_RPC.String(), messageIDs)
		require.NoError(t, err)

		// Verify that we get results (exact count depends on sharding)
		assert.LessOrEqual(t, len(result), len(messageIDs), "Should not get more results than input messages")
		t.Logf("Message IDs format: filtered %d out of %d messages", len(result), len(messageIDs))
	})

	// Test with invalid type
	t.Run("unified_interface_invalid_type", func(t *testing.T) {
		invalidMessages := []string{"invalid", "type"}

		result, err := mimicry.ShouldTraceRPCMetaMessages(event, clientMeta, xatu.Event_LIBP2P_TRACE_RECV_RPC.String(), invalidMessages)
		assert.Error(t, err, "Should return error for invalid message type")
		assert.Nil(t, result, "Should return nil result on error")
		assert.Contains(t, err.Error(), "unsupported message type", "Error should mention unsupported type")
	})
}

// TestBackwardCompatibilityWithHierarchical verifies that simple configs still work alongside hierarchical ones
func TestBackwardCompatibilityWithHierarchical(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	testTime := time.Now()
	uint64Ptr := uint64(64)

	// Create mixed configuration (simple + hierarchical)
	config := &Config{
		Traces: TracesConfig{
			Enabled: true,
			Topics: map[string]TopicConfig{
				// Simple configuration
				"(?i).*add_peer.*": {
					TotalShards:     &uint64Ptr,
					ActiveShardsRaw: &ActiveShardsConfig{0, 1, 2},
					ShardingKey:     "PeerID",
				},
				// Hierarchical configuration
				"(?i).*duplicate_message.*": {
					Topics: &TopicsConfig{
						Fallback: &GossipTopicConfig{
							TotalShards:     8,
							ActiveShardsRaw: ActiveShardsConfig{0},
							ShardingKey:     "MsgID",
						},
					},
				},
			},
		},
	}

	// Validate and compile the configuration
	err := config.Traces.Validate()
	require.NoError(t, err)
	err = config.Traces.CompilePatterns()
	require.NoError(t, err)

	mimicry := &Mimicry{
		Config:  config,
		metrics: NewMetrics("test_backward_compat"),
	}

	clientMeta := createMockClientMeta(1) // mainnet

	// Test simple configuration
	addPeerEvent := &host.TraceEvent{
		Type:      "ADD_PEER",
		PeerID:    testPeerID,
		Timestamp: testTime,
		Payload:   map[string]any{"PeerID": testPeerID.String()},
	}

	simpleResult := mimicry.ShouldTraceMessage(
		addPeerEvent,
		clientMeta,
		xatu.Event_LIBP2P_TRACE_ADD_PEER.String(),
	)

	// Calculate expected result for simple config
	expectedShard := GetShard(testPeerID.String(), 64)
	expectedSimpleResult := expectedShard <= 2
	assert.Equal(t, expectedSimpleResult, simpleResult, "Simple configuration should work as before")

	// Test hierarchical configuration
	duplicateEvent := &host.TraceEvent{
		Type:      "DUPLICATE_MESSAGE",
		PeerID:    testPeerID,
		Timestamp: testTime,
		Payload: map[string]any{
			"MsgID": "test_msg_hierarchical",
			"Topic": "/eth2/4a26c58b/some_topic/ssz_snappy",
		},
	}

	hierarchicalResult := mimicry.ShouldTraceMessage(
		duplicateEvent,
		clientMeta,
		xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String(),
	)

	// Calculate expected result for hierarchical config (fallback)
	expectedHierarchicalShard := GetShard("test_msg_hierarchical", 8)
	expectedHierarchicalResult := expectedHierarchicalShard == 0
	assert.Equal(t, expectedHierarchicalResult, hierarchicalResult, "Hierarchical configuration should work correctly")

	t.Logf("Simple config result: %v, Hierarchical config result: %v", simpleResult, hierarchicalResult)
}
