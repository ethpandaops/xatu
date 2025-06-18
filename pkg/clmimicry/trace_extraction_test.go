package clmimicry

import (
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/probe-lab/hermes/host"
	"github.com/stretchr/testify/assert"
)

// TestGetShardingKey validates the extraction of sharding keys from various
// libp2p trace event types and payload structures.
//
// Sharding keys are the critical values used for consistent hash-based routing:
// - MsgID: Uses message identifier for message-based sharding
// - PeerID: Uses peer identifier for peer-based sharding
//
// This test ensures proper key extraction across:
// 1. Different event types (CONNECTED, RECV_RPC, DUPLICATE_MESSAGE, etc.)
// 2. Various payload formats (maps, protobuf, RPC structures)
// 3. Edge cases (missing fields, wrong types, empty values)
//
// Correct key extraction is essential for deterministic shard assignment.
func TestGetShardingKey(t *testing.T) {
	var (
		remotePeerIDStr = "16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq"
		testPeerID, _   = peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
		testTime        = time.Now()
		mockMsgID       = "test-msg-id"
	)

	testCases := []struct {
		name            string
		event           *host.TraceEvent
		eventType       string
		shardingKeyType string
		expectedKey     string
	}{
		{
			name: "MsgID sharding key",
			event: &host.TraceEvent{
				Type:      "TEST_EVENT",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload:   &MockPayload{MsgID: mockMsgID},
			},
			eventType:       "TEST_EVENT",
			shardingKeyType: string(ShardingKeyTypeMsgID),
			expectedKey:     mockMsgID,
		},
		{
			name: "PeerID sharding key with payload lacking PeerID",
			event: &host.TraceEvent{
				Type:      "TEST_EVENT",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload:   &MockPayload{MsgID: mockMsgID},
			},
			eventType:       "TEST_EVENT",
			shardingKeyType: string(ShardingKeyTypePeerID),
			expectedKey:     testPeerID.String(),
		},
		{
			name: "PeerID sharding key with map payload containing PeerID",
			event: &host.TraceEvent{
				Type:      "TEST_EVENT",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: map[string]any{
					"PeerID": remotePeerIDStr,
					"Other":  "value",
				},
			},
			eventType:       "TEST_EVENT",
			shardingKeyType: string(ShardingKeyTypePeerID),
			expectedKey:     remotePeerIDStr,
		},
		{
			name: "Default to MsgID for unknown type",
			event: &host.TraceEvent{
				Type:      "TEST_EVENT",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload:   &MockPayload{MsgID: mockMsgID},
			},
			eventType:       "TEST_EVENT",
			shardingKeyType: "UnknownType",
			expectedKey:     mockMsgID,
		},
		{
			name: "Empty payload returns empty string for MsgID",
			event: &host.TraceEvent{
				Type:      "TEST_EVENT",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload:   nil,
			},
			eventType:       "TEST_EVENT",
			shardingKeyType: string(ShardingKeyTypeMsgID),
			expectedKey:     "",
		},
		// Add tests for special event types
		{
			name: "PeerID sharding with ADD_PEER event",
			event: &host.TraceEvent{
				Type:   pubsubpb.TraceEvent_ADD_PEER.String(),
				PeerID: testPeerID,
				Payload: map[string]any{
					"PeerID":   testPeerID,
					"Protocol": "test-protocol",
				},
			},
			eventType:       xatu.Event_LIBP2P_TRACE_ADD_PEER.String(),
			shardingKeyType: string(ShardingKeyTypePeerID),
			expectedKey:     testPeerID.String(), // Should get PeerID from the converter
		},
		{
			name: "Unshardable JOIN event",
			event: &host.TraceEvent{
				Type:   pubsubpb.TraceEvent_JOIN.String(),
				PeerID: testPeerID,
				Payload: map[string]any{
					"Topic": "test-topic",
				},
			},
			eventType:       xatu.Event_LIBP2P_TRACE_JOIN.String(),
			shardingKeyType: string(ShardingKeyTypePeerID),
			expectedKey:     testPeerID.String(), // No PeerID in JOIN, should fallback to host peer
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetShardingKey(tc.event, &xatu.ClientMeta{}, tc.shardingKeyType, tc.eventType)
			assert.Equal(t, tc.expectedKey, result)
		})
	}
}

// TestGetMsgID validates message ID extraction from various payload structures
// using Go reflection to handle different data types dynamically.
//
// Message IDs are essential for consistent message-based sharding and must be
// extracted reliably from diverse payload formats:
// 1. Struct pointers with MsgID fields
// 2. Map[string]interface{} with "MsgID" keys
// 3. Map[string]string with string values
// 4. Missing or nil payloads (should return empty string)
//
// The extraction logic uses reflection to handle the dynamic nature of
// libp2p trace event payloads while maintaining type safety.
func TestGetMsgID(t *testing.T) {
	mockMsgID := "test-msg-id-12345"

	testCases := []struct {
		name        string
		payload     interface{}
		expectedKey string
	}{
		{
			name:        "Struct pointer payload with MsgID",
			payload:     &MockPayload{MsgID: mockMsgID},
			expectedKey: mockMsgID,
		},
		{
			name: "Map payload with MsgID (deliver_message/duplicate_message style)",
			payload: map[string]any{
				"MsgID":   mockMsgID,
				"Topic":   "test-topic",
				"PeerID":  "peer-123",
				"Local":   true,
				"MsgSize": 1024,
			},
			expectedKey: mockMsgID,
		},
		{
			name:        "Nil payload returns empty string",
			payload:     nil,
			expectedKey: "",
		},
		{
			name: "Map payload missing MsgID returns empty string",
			payload: map[string]any{
				"Topic":  "test-topic",
				"PeerID": "peer-123",
			},
			expectedKey: "",
		},
		{
			name: "Map payload with non-string MsgID returns empty string",
			payload: map[string]any{
				"MsgID": 12345, // Not a string
				"Topic": "test-topic",
			},
			expectedKey: "",
		},
		{
			name:        "Struct pointer with empty MsgID",
			payload:     &MockPayload{MsgID: ""},
			expectedKey: "",
		},
		{
			name:        "Non-struct, non-map payload returns empty string",
			payload:     []string{"not", "a", "struct", "or", "map"},
			expectedKey: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getMsgID(tc.payload)
			assert.Equal(t, tc.expectedKey, result)
		})
	}
}

// MockStringWrapper mimics *wrapperspb.StringValue behavior for testing
type MockStringWrapper struct {
	Value string
}

func (m *MockStringWrapper) GetValue() string {
	return m.Value
}

// MockPayloadWithTopic mimics a protobuf struct with a Topic field
type MockPayloadWithTopic struct {
	Topic *MockStringWrapper
}

// TestGetGossipTopics validates extraction of multiple gossip topics from RPC events
// for multi-topic support in hierarchical sharding decisions.
//
// RPC events (recv_rpc, send_rpc, drop_rpc) can contain multiple topics in:
// - Messages: Each message can have a different topic
// - Subscriptions: Each subscription has a TopicID
// - Control Messages: IHave, Graft, Prune each contain TopicID fields
//
// This test ensures reliable extraction of ALL topics for pattern matching.
func TestGetGossipTopics(t *testing.T) {
	testPeerID, _ := peer.Decode("16Uiu2HAmPjTC9u4nSvufM2weykDx7aYK3SiHoXCqngk3vJ2TR229")
	testTime := time.Now()

	testCases := []struct {
		name           string
		event          *host.TraceEvent
		expectedTopics []string
	}{
		{
			name:           "nil event returns nil",
			event:          nil,
			expectedTopics: nil,
		},
		{
			name: "event with nil payload returns empty slice",
			event: &host.TraceEvent{
				Type:      "TEST_EVENT",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload:   nil,
			},
			expectedTopics: []string{},
		},
		{
			name: "map payload with single Topic field",
			event: &host.TraceEvent{
				Type:      "TEST_EVENT",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: map[string]any{
					"Topic": "/eth2/4a26c58b/beacon_block/ssz_snappy",
					"MsgID": "test-msg-id",
				},
			},
			expectedTopics: []string{"/eth2/4a26c58b/beacon_block/ssz_snappy"},
		},
		{
			name: "RPC meta with multiple message topics",
			event: &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: &host.RpcMeta{
					PeerID: testPeerID,
					Messages: []host.RpcMetaMsg{
						{Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy", MsgID: "msg1"},
						{Topic: "/eth2/4a26c58b/beacon_attestation_1/ssz_snappy", MsgID: "msg2"},
						{Topic: "/eth2/4a26c58b/blob_sidecar_0/ssz_snappy", MsgID: "msg3"},
					},
				},
			},
			expectedTopics: []string{
				"/eth2/4a26c58b/beacon_block/ssz_snappy",
				"/eth2/4a26c58b/beacon_attestation_1/ssz_snappy",
				"/eth2/4a26c58b/blob_sidecar_0/ssz_snappy",
			},
		},
		{
			name: "RPC meta with subscription topics",
			event: &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: &host.RpcMeta{
					PeerID: testPeerID,
					Subscriptions: []host.RpcMetaSub{
						{TopicID: "/eth2/4a26c58b/beacon_block/ssz_snappy", Subscribe: true},
						{TopicID: "/eth2/4a26c58b/voluntary_exit/ssz_snappy", Subscribe: false},
					},
				},
			},
			expectedTopics: []string{
				"/eth2/4a26c58b/beacon_block/ssz_snappy",
				"/eth2/4a26c58b/voluntary_exit/ssz_snappy",
			},
		},
		{
			name: "RPC meta with control message topics",
			event: &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: &host.RpcMeta{
					PeerID: testPeerID,
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{TopicID: "/eth2/4a26c58b/beacon_attestation_5/ssz_snappy", MsgIDs: []string{"ihave1"}},
						},
						Graft: []host.RpcControlGraft{
							{TopicID: "/eth2/4a26c58b/beacon_block/ssz_snappy"},
						},
						Prune: []host.RpcControlPrune{
							{TopicID: "/eth2/4a26c58b/blob_sidecar_2/ssz_snappy"},
						},
					},
				},
			},
			expectedTopics: []string{
				"/eth2/4a26c58b/beacon_attestation_5/ssz_snappy",
				"/eth2/4a26c58b/beacon_block/ssz_snappy",
				"/eth2/4a26c58b/blob_sidecar_2/ssz_snappy",
			},
		},
		{
			name: "RPC meta with mixed topic sources and duplicates",
			event: &host.TraceEvent{
				Type:      "SEND_RPC",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: &host.RpcMeta{
					PeerID: testPeerID,
					Messages: []host.RpcMetaMsg{
						{Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy", MsgID: "msg1"},
						{Topic: "/eth2/4a26c58b/beacon_attestation_1/ssz_snappy", MsgID: "msg2"},
					},
					Subscriptions: []host.RpcMetaSub{
						{TopicID: "/eth2/4a26c58b/beacon_block/ssz_snappy", Subscribe: true}, // Duplicate
						{TopicID: "/eth2/4a26c58b/voluntary_exit/ssz_snappy", Subscribe: false},
					},
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{TopicID: "/eth2/4a26c58b/beacon_attestation_1/ssz_snappy", MsgIDs: []string{"ihave1"}}, // Duplicate
						},
						Graft: []host.RpcControlGraft{
							{TopicID: "/eth2/4a26c58b/blob_sidecar_3/ssz_snappy"},
						},
					},
				},
			},
			expectedTopics: []string{
				"/eth2/4a26c58b/beacon_block/ssz_snappy",
				"/eth2/4a26c58b/beacon_attestation_1/ssz_snappy",
				"/eth2/4a26c58b/voluntary_exit/ssz_snappy",
				"/eth2/4a26c58b/blob_sidecar_3/ssz_snappy",
			},
		},
		{
			name: "RPC meta with empty topic fields",
			event: &host.TraceEvent{
				Type:      "DROP_RPC",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: &host.RpcMeta{
					PeerID: testPeerID,
					Messages: []host.RpcMetaMsg{
						{Topic: "", MsgID: "msg1"}, // Empty topic should be ignored
						{Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy", MsgID: "msg2"},
					},
					Subscriptions: []host.RpcMetaSub{
						{TopicID: "", Subscribe: true}, // Empty topic should be ignored
					},
				},
			},
			expectedTopics: []string{"/eth2/4a26c58b/beacon_block/ssz_snappy"},
		},
		{
			name: "RPC meta with no topics",
			event: &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload: &host.RpcMeta{
					PeerID:        testPeerID,
					Messages:      []host.RpcMetaMsg{},
					Subscriptions: []host.RpcMetaSub{},
					Control:       nil,
				},
			},
			expectedTopics: []string{},
		},
		{
			name: "struct payload with Topic field (backward compatibility)",
			event: &host.TraceEvent{
				Type:      "TEST_EVENT",
				PeerID:    testPeerID,
				Timestamp: testTime,
				Payload:   &MockPayloadWithTopic{Topic: &MockStringWrapper{Value: "/eth2/4a26c58b/beacon_attestation_1/ssz_snappy"}},
			},
			expectedTopics: []string{"/eth2/4a26c58b/beacon_attestation_1/ssz_snappy"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetGossipTopics(tc.event)

			// Sort both slices for comparison since map iteration order is not guaranteed
			if len(result) > 0 && len(tc.expectedTopics) > 0 {
				assert.ElementsMatch(t, tc.expectedTopics, result, "Topics should match regardless of order")
			} else {
				assert.Equal(t, len(tc.expectedTopics), len(result), "Should have same number of topics")
			}
		})
	}
}
