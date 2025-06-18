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

// TestHierarchicalTopicConfig tests the new hierarchical gossip topic configuration
func TestHierarchicalTopicConfig(t *testing.T) {
	uint64Ptr := uint64(64)

	tests := []struct {
		name      string
		config    TracesConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid hierarchical config with gossip topics",
			config: TracesConfig{
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
									ActiveShardsRaw: ActiveShardsConfig{0, 1},
									ShardingKey:     "PeerID",
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
			wantError: false,
		},
		{
			name: "valid hierarchical config with only fallback",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*deliver_message.*": {
						Topics: &TopicsConfig{
							Fallback: &GossipTopicConfig{
								TotalShards:     64,
								ActiveShardsRaw: ActiveShardsConfig{"0-3"},
								ShardingKey:     "MsgID",
							},
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "invalid hierarchical config - both simple and hierarchical",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						TotalShards:     &uint64Ptr,
						ActiveShardsRaw: &ActiveShardsConfig{0, 1},
						Topics: &TopicsConfig{
							Fallback: &GossipTopicConfig{
								TotalShards:     64,
								ActiveShardsRaw: ActiveShardsConfig{0},
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "cannot use both simple and hierarchical configuration",
		},
		{
			name: "invalid hierarchical config - no gossip topics or fallback",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						Topics: &TopicsConfig{},
					},
				},
			},
			wantError: true,
			errorMsg:  "must have either gossipTopics or fallback",
		},
		{
			name: "invalid gossip topic regex pattern",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						Topics: &TopicsConfig{
							GossipTopics: map[string]GossipTopicConfig{
								"[unclosed": {
									TotalShards:     1,
									ActiveShardsRaw: ActiveShardsConfig{0},
								},
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "invalid gossip topic pattern",
		},
		{
			name: "invalid gossip topic config - zero total shards",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						Topics: &TopicsConfig{
							GossipTopics: map[string]GossipTopicConfig{
								".*beacon_block.*": {
									TotalShards:     0,
									ActiveShardsRaw: ActiveShardsConfig{0},
								},
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "totalShards must be greater than 0",
		},
		{
			name: "invalid gossip topic config - empty active shards",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						Topics: &TopicsConfig{
							GossipTopics: map[string]GossipTopicConfig{
								".*beacon_block.*": {
									TotalShards:     4,
									ActiveShardsRaw: ActiveShardsConfig{},
								},
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "activeShards cannot be empty",
		},
		{
			name: "invalid gossip topic config - active shard out of range",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						Topics: &TopicsConfig{
							GossipTopics: map[string]GossipTopicConfig{
								".*beacon_block.*": {
									TotalShards:     4,
									ActiveShardsRaw: ActiveShardsConfig{0, 5}, // 5 is out of range [0-3]
								},
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "active shard 5 is out of range",
		},
		{
			name: "invalid sharding key",
			config: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*duplicate_message.*": {
						Topics: &TopicsConfig{
							GossipTopics: map[string]GossipTopicConfig{
								".*beacon_block.*": {
									TotalShards:     1,
									ActiveShardsRaw: ActiveShardsConfig{0},
									ShardingKey:     "InvalidKey",
								},
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "invalid sharding key 'InvalidKey'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)

				// Verify that active shards were processed correctly
				for pattern, topicConfig := range tt.config.Topics {
					if topicConfig.Topics != nil {
						if topicConfig.Topics.GossipTopics != nil {
							for gossipPattern, gossipConfig := range topicConfig.Topics.GossipTopics {
								assert.NotEmpty(t, gossipConfig.ActiveShards,
									"Active shards should be processed for pattern %s, gossip pattern %s", pattern, gossipPattern)
							}
						}
						if topicConfig.Topics.Fallback != nil {
							assert.NotEmpty(t, topicConfig.Topics.Fallback.ActiveShards,
								"Active shards should be processed for fallback in pattern %s", pattern)
						}
					}
				}
			}
		})
	}
}

// TestHierarchicalConfigCompilation tests that hierarchical configs compile correctly
func TestHierarchicalConfigCompilation(t *testing.T) {
	uint64Ptr := uint64(64)

	config := TracesConfig{
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
							ActiveShardsRaw: ActiveShardsConfig{0, 1},
							ShardingKey:     "PeerID",
						},
					},
					Fallback: &GossipTopicConfig{
						TotalShards:     512,
						ActiveShardsRaw: ActiveShardsConfig{0},
						ShardingKey:     "MsgID",
					},
				},
			},
			"(?i).*simple_event.*": {
				TotalShards:     &uint64Ptr,
				ActiveShardsRaw: &ActiveShardsConfig{0, 1, 2},
				ShardingKey:     "MsgID",
			},
		},
	}

	// Validate the config first
	err := config.Validate()
	require.NoError(t, err)

	// Compile patterns
	err = config.CompilePatterns()
	require.NoError(t, err)

	// Check that patterns were compiled
	assert.Len(t, config.compiledPatterns, 2)
	assert.Len(t, config.compiledTopics, 2)

	// Check that gossip topic patterns were compiled
	for pattern, compiledConfig := range config.compiledTopics {
		originalConfig := config.Topics[pattern]
		assert.Equal(t, &originalConfig, compiledConfig.Original)

		if originalConfig.Topics != nil && originalConfig.Topics.GossipTopics != nil {
			assert.Len(t, compiledConfig.GossipPatterns, len(originalConfig.Topics.GossipTopics))
		} else {
			assert.Empty(t, compiledConfig.GossipPatterns)
		}
	}
}

// TestHierarchicalLogSummary tests the log summary for hierarchical configurations
func TestHierarchicalLogSummary(t *testing.T) {
	uint64Ptr := uint64(64)

	config := TracesConfig{
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
							ActiveShardsRaw: ActiveShardsConfig{0, 1},
							ShardingKey:     "PeerID",
						},
					},
					Fallback: &GossipTopicConfig{
						TotalShards:     512,
						ActiveShardsRaw: ActiveShardsConfig{0},
						ShardingKey:     "MsgID",
					},
				},
			},
			"(?i).*simple_event.*": {
				TotalShards:     &uint64Ptr,
				ActiveShardsRaw: &ActiveShardsConfig{0, 1, 2},
				ShardingKey:     "MsgID",
			},
		},
	}

	// Validate first
	err := config.Validate()
	require.NoError(t, err)

	summary := config.LogSummary()

	// Check for hierarchical pattern indication
	assert.Contains(t, summary, "HIERARCHICAL")

	// Check for gossip topic patterns
	assert.Contains(t, summary, "Gossip Topics:")
	assert.Contains(t, summary, ".*beacon_block.*")
	assert.Contains(t, summary, ".*beacon_attestation.*")

	// Check for fallback
	assert.Contains(t, summary, "Fallback:")

	// Check for simple config
	assert.Contains(t, summary, "(?i).*simple_event.*")
	assert.Contains(t, summary, "3/64 shards active")

	// Check sharding key information
	assert.Contains(t, summary, "sharding on MsgID")
	assert.Contains(t, summary, "sharding on PeerID")
}

// Note: TestHierarchicalShardingIntegration removed - covered by TestComprehensiveEventSharding in trace_integration_test.go

// Note: TestHierarchicalShardingWithProtobufPayload removed - covered by TestComprehensiveEventSharding in trace_integration_test.go

// Note: TestHierarchicalRPCMetaFiltering removed - covered by TestComprehensiveEventSharding in trace_integration_test.go

// Note: TestHierarchicalRPCMetaMessageFiltering removed - covered by TestComprehensiveEventSharding in trace_integration_test.go
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

// TestRPCMetaControlHierarchicalSharding tests that RPC meta control messages (IHAVE, GRAFT, PRUNE)
// now properly support hierarchical sharding using their topic information.
func TestRPCMetaControlHierarchicalSharding(t *testing.T) {
	tests := []struct {
		name             string
		messageType      string
		topicValue       string
		expectedSampling bool
		testDescription  string
	}{
		{
			name:             "IHAVE with beacon_block topic - 100% sampling",
			messageType:      "IHAVE",
			topicValue:       "/eth2/4a26c58b/beacon_block/ssz_snappy",
			expectedSampling: true,
			testDescription:  "IHAVE messages for beacon_block topic should be sampled at 100%",
		},
		{
			name:             "IHAVE with beacon_attestation topic - 50% sampling",
			messageType:      "IHAVE",
			topicValue:       "/eth2/4a26c58b/beacon_attestation_1/ssz_snappy",
			expectedSampling: false, // This specific message ID should be filtered out
			testDescription:  "IHAVE messages for beacon_attestation topic should use 50% sampling",
		},
		{
			name:             "GRAFT with beacon_block topic - 100% sampling",
			messageType:      "GRAFT",
			topicValue:       "/eth2/4a26c58b/beacon_block/ssz_snappy",
			expectedSampling: true,
			testDescription:  "GRAFT messages for beacon_block topic should be sampled at 100%",
		},
		{
			name:             "PRUNE with beacon_attestation topic - 50% sampling",
			messageType:      "PRUNE",
			topicValue:       "/eth2/4a26c58b/beacon_attestation_1/ssz_snappy",
			expectedSampling: false,
			testDescription:  "PRUNE messages for beacon_attestation topic should use 50% sampling",
		},
		{
			name:             "IHAVE with fallback topic - 1/512 sampling",
			messageType:      "IHAVE",
			topicValue:       "/eth2/4a26c58b/voluntary_exit/ssz_snappy",
			expectedSampling: false,
			testDescription:  "IHAVE messages for unmatched topics should use fallback 1/512 sampling",
		},
	}

	// Create test peer ID
	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create hierarchical configuration for RPC meta events
			config := &Config{
				Traces: TracesConfig{
					Enabled: true,
					Topics: map[string]TopicConfig{
						// IHAVE events
						"(?i).*ihave.*": {
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
						// GRAFT events
						"(?i).*graft.*": {
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
						// PRUNE events
						"(?i).*prune.*": {
							Topics: &TopicsConfig{
								GossipTopics: map[string]GossipTopicConfig{
									".*beacon_block.*": {
										TotalShards:     1,
										ActiveShardsRaw: ActiveShardsConfig{0}, // 100% sampling
										ShardingKey:     "PeerID",
									},
									".*beacon_attestation.*": {
										TotalShards:     4,
										ActiveShardsRaw: ActiveShardsConfig{0, 1}, // 50% sampling
										ShardingKey:     "PeerID",
									},
								},
								Fallback: &GossipTopicConfig{
									TotalShards:     512,
									ActiveShardsRaw: ActiveShardsConfig{0}, // 1/512 sampling
									ShardingKey:     "PeerID",
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
				metrics: NewMetrics("test_rpc_meta_" + strings.ReplaceAll(tt.name, " ", "_")),
			}

			var result bool

			switch tt.messageType {
			case "IHAVE":
				// Test IHAVE message filtering
				messageInfos := []RPCMetaMessageInfo{
					{
						MessageID: wrapperspb.String("test_msg_12345"),
						Topic:     wrapperspb.String(tt.topicValue),
					},
				}

				event := &host.TraceEvent{
					PeerID:    testPeerID,
					Timestamp: time.Now(),
					Payload:   nil, // Not used for IHAVE filtering
				}

				clientMeta := createMockClientMeta(1) // mainnet

				filteredMessages, err := mimicry.ShouldTraceRPCMetaMessages(
					event,
					clientMeta,
					xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE.String(),
					messageInfos,
				)
				assert.NoError(t, err, "IHAVE filtering should not error")
				result = len(filteredMessages) > 0

			case "GRAFT":
				// Test GRAFT event filtering by creating a mock event with topic
				graftEvent := &host.TraceEvent{
					PeerID:    testPeerID,
					Timestamp: time.Now(),
					Payload: map[string]any{
						"Topic": tt.topicValue,
					},
				}

				clientMeta := createMockClientMeta(1) // mainnet
				result = mimicry.ShouldTraceMessage(graftEvent, clientMeta, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT.String())

			case "PRUNE":
				// Test PRUNE peer filtering
				peerInfos := []RPCMetaPeerInfo{
					{
						PeerID: wrapperspb.String("16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq"),
						Topic:  wrapperspb.String(tt.topicValue),
					},
				}

				event := &host.TraceEvent{
					PeerID:    testPeerID,
					Timestamp: time.Now(),
					Payload:   nil, // Not used for PRUNE filtering
				}

				clientMeta := createMockClientMeta(1) // mainnet

				filteredPeers, err := mimicry.ShouldTraceRPCMetaMessages(
					event,
					clientMeta,
					xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE.String(),
					peerInfos,
				)
				assert.NoError(t, err, "PRUNE filtering should not error")
				result = len(filteredPeers) > 0
			}

			assert.Equal(t, tt.expectedSampling, result, tt.testDescription)
			t.Logf("Test: %s, Topic: %s, Expected: %v, Result: %v",
				tt.name, tt.topicValue, tt.expectedSampling, result)
		})
	}
}

// TestHierarchicalNoFallbackConfig tests hierarchical config scenarios
func TestHierarchicalNoFallbackConfig(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	t.Run("no_fallback_config_found", func(t *testing.T) {
		config := &Config{
			Traces: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*recv_rpc.*": {
						Topics: &TopicsConfig{
							GossipTopics: map[string]GossipTopicConfig{
								".*specific_topic.*": {
									TotalShards:     4,
									ActiveShardsRaw: ActiveShardsConfig{0, 1},
									ShardingKey:     "MsgID",
								},
							},
							// No Fallback config - this should trigger the "no fallback" path
							Fallback: nil,
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
			metrics: NewMetrics("test_no_fallback"),
		}

		testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
		event := &host.TraceEvent{
			Type:      "RECV_RPC",
			PeerID:    testPeerID,
			Timestamp: time.Now(),
			Payload:   map[string]any{},
		}
		clientMeta := createMockClientMeta(1)

		// Test with message IDs (no topic info) - should use fallback but none exists
		messageIDs := []*wrapperspb.StringValue{
			wrapperspb.String("msg_1"),
			wrapperspb.String("msg_2"),
		}

		result, err := mimicry.ShouldTraceRPCMetaMessages(
			event,
			clientMeta,
			xatu.Event_LIBP2P_TRACE_RECV_RPC.String(),
			messageIDs,
		)

		require.NoError(t, err)
		assert.Empty(t, result, "Should return empty result when no fallback config exists")
	})

	t.Run("no_configuration_found_hierarchical", func(t *testing.T) {
		config := &Config{
			Traces: TracesConfig{
				Enabled: true,
				Topics: map[string]TopicConfig{
					"(?i).*recv_rpc.*": {
						Topics: &TopicsConfig{
							GossipTopics: map[string]GossipTopicConfig{
								".*specific_topic.*": {
									TotalShards:     4,
									ActiveShardsRaw: ActiveShardsConfig{0, 1},
									ShardingKey:     "MsgID",
								},
							},
							// No Fallback config
							Fallback: nil,
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
			metrics: NewMetrics("test_no_config_hierarchical"),
		}

		testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
		event := &host.TraceEvent{
			Type:      "RECV_RPC",
			PeerID:    testPeerID,
			Timestamp: time.Now(),
			Payload: map[string]any{
				"Topic": "/eth2/unmatched/topic", // This won't match any gossip topic pattern
			},
		}
		clientMeta := createMockClientMeta(1)

		// Test shouldTraceWithHierarchicalConfig directly
		topicConfig, found := config.Traces.FindMatchingTopicConfig(xatu.Event_LIBP2P_TRACE_RECV_RPC.String())
		require.True(t, found)

		result := mimicry.shouldTraceWithHierarchicalConfig(event, clientMeta, topicConfig, xatu.Event_LIBP2P_TRACE_RECV_RPC.String())
		assert.False(t, result, "Should return false when no configuration found in hierarchical mode")

		// Test with RPCMetaMessageInfo with unmatched topic
		messages := []RPCMetaMessageInfo{
			{
				MessageID: wrapperspb.String("msg_1"),
				Topic:     wrapperspb.String("/eth2/unmatched/topic"),
			},
		}

		filteredResult, err := mimicry.ShouldTraceRPCMetaMessages(
			event,
			clientMeta,
			xatu.Event_LIBP2P_TRACE_RECV_RPC.String(),
			messages,
		)

		require.NoError(t, err)
		assert.Empty(t, filteredResult, "Should return empty when messages have unmatched topics and no fallback")
	})
}
