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
