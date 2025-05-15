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
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
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
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
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
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
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
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
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
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
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
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
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
								TotalShards: 4,
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

// TestGetShard verifies that the GetShard function consistently maps message IDs to shards.
func TestGetShard(t *testing.T) {
	tests := []struct {
		msgID       string
		totalShards uint64
		expected    uint64
	}{
		{
			msgID:       "0x1234",
			totalShards: 64,
			expected:    GetShard("0x1234", 64),
		},
		{
			msgID:       "0xabcd",
			totalShards: 64,
			expected:    GetShard("0xabcd", 64),
		},
		{
			msgID:       "0x1234",
			totalShards: 128,
			expected:    GetShard("0x1234", 128),
		},
	}

	for _, tt := range tests {
		t.Run(tt.msgID, func(t *testing.T) {
			// Call it multiple times to verify consistency
			for i := 0; i < 5; i++ {
				result := GetShard(tt.msgID, tt.totalShards)
				assert.Equal(t, tt.expected, result)
				assert.Less(t, result, tt.totalShards)
			}
		})
	}
}

// TestIsShardActive tests the IsShardActive function
func TestIsShardActive(t *testing.T) {
	tests := []struct {
		name         string
		shard        uint64
		activeShards []uint64
		expected     bool
	}{
		{
			name:         "shard is active",
			shard:        2,
			activeShards: []uint64{1, 2, 3},
			expected:     true,
		},
		{
			name:         "shard is not active",
			shard:        4,
			activeShards: []uint64{1, 2, 3},
			expected:     false,
		},
		{
			name:         "empty active shards",
			shard:        1,
			activeShards: []uint64{},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsShardActive(tt.shard, tt.activeShards)
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
						TotalShards:  totalShards,
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
						TotalShards:  totalShards,
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

// TestShouldTraceMessageWithDifferentShardingKeys tests the ShouldTraceMessage function
// with different sharding key configurations.
func TestShouldTraceMessageWithDifferentShardingKeys(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	testPeerID, _ := peer.Decode("QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	testTime := time.Now()
	testMsgID := "test-msg-id"

	// Create a consistent event to test different sharding keys
	event := &host.TraceEvent{
		Type:      "TEST_EVENT",
		Topic:     "test-topic",
		PeerID:    testPeerID,
		Timestamp: testTime,
		Payload:   &MockPayload{MsgID: testMsgID},
	}

	clientMeta := createMockClientMeta(networks.DeriveFromID(1).ID) // mainnet

	// Create test cases with different sharding key configurations
	tests := []struct {
		name         string
		topicConfig  TopicConfig
		expectedTrue bool
	}{
		{
			name: "MsgID sharding with active shard",
			topicConfig: TopicConfig{
				TotalShards:  64,
				ActiveShards: []uint64{GetShard(testMsgID, 64)}, // The shard for testMsgID
				ShardingKey:  string(ShardingKeyTypeMsgID),
			},
			expectedTrue: true,
		},
		{
			name: "MsgID sharding with inactive shard",
			topicConfig: TopicConfig{
				TotalShards:  64,
				ActiveShards: []uint64{(GetShard(testMsgID, 64) + 1) % 64}, // A different shard
				ShardingKey:  string(ShardingKeyTypeMsgID),
			},
			expectedTrue: false,
		},
		{
			name: "PeerID sharding with active shard",
			topicConfig: TopicConfig{
				TotalShards:  64,
				ActiveShards: []uint64{GetShard(testPeerID.String(), 64)}, // The shard for testPeerID
				ShardingKey:  string(ShardingKeyTypePeerID),
			},
			expectedTrue: true,
		},
		{
			name: "PeerID sharding with inactive shard",
			topicConfig: TopicConfig{
				TotalShards:  64,
				ActiveShards: []uint64{(GetShard(testPeerID.String(), 64) + 1) % 64}, // A different shard
				ShardingKey:  string(ShardingKeyTypePeerID),
			},
			expectedTrue: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mimicry := &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"TEST_EVENT": tt.topicConfig,
						},
					},
				},
				metrics: NewMetrics(fmt.Sprintf("test_sharding_key_%d", i)),
			}

			// Compile the regex patterns
			err := mimicry.Config.Traces.CompilePatterns()
			assert.NoError(t, err)

			result := mimicry.ShouldTraceMessage(event, clientMeta, "TEST_EVENT")
			assert.Equal(t, tt.expectedTrue, result)
		})
	}
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
