package clmimicry_test

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clmimicry"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/probe-lab/hermes/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestGetMsgID(t *testing.T) {
	t.Run("from map payload", func(t *testing.T) {
		tests := []struct {
			name     string
			payload  map[string]any
			expected string
		}{
			{
				name: "valid MsgID",
				payload: map[string]any{
					"MsgID": "msg-123",
					"Topic": "test-topic",
				},
				expected: "msg-123",
			},
			{
				name: "MsgID not string",
				payload: map[string]any{
					"MsgID": 123, // not a string
				},
				expected: "",
			},
			{
				name:     "no MsgID field",
				payload:  map[string]any{"Topic": "test-topic"},
				expected: "",
			},
			{
				name:     "empty map",
				payload:  map[string]any{},
				expected: "",
			},
			{
				name:     "nil map",
				payload:  nil,
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				event := &host.TraceEvent{
					Payload: tt.payload,
				}
				result := clmimicry.GetMsgID(event)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("from struct payload via reflection", func(t *testing.T) {
		// Create a test struct that mimics protobuf structs with MsgID field
		type TestPayload struct {
			MsgID string
			Topic string
		}

		event := &host.TraceEvent{
			Payload: &TestPayload{
				MsgID: "struct-msg-id",
				Topic: "struct-topic",
			},
		}

		result := clmimicry.GetMsgID(event)
		assert.Equal(t, "struct-msg-id", result)
	})

	t.Run("edge cases", func(t *testing.T) {
		tests := []struct {
			name     string
			event    *host.TraceEvent
			expected string
		}{
			{
				name:     "nil event",
				event:    nil,
				expected: "",
			},
			{
				name:     "nil payload",
				event:    &host.TraceEvent{Payload: nil},
				expected: "",
			},
			{
				name:     "non-pointer struct payload",
				event:    &host.TraceEvent{Payload: struct{ MsgID string }{MsgID: "test"}},
				expected: "",
			},
			{
				name:     "string payload",
				event:    &host.TraceEvent{Payload: "not-a-struct"},
				expected: "",
			},
			{
				name:     "int payload",
				event:    &host.TraceEvent{Payload: 123},
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := clmimicry.GetMsgID(tt.event)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestGetGossipTopics(t *testing.T) {
	t.Run("from RpcMeta payload", func(t *testing.T) {
		peerID, _ := peer.Decode("16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq")

		event := &host.TraceEvent{
			Type:   "LIBP2P_TRACE_RECV_RPC",
			PeerID: peerID,
			Payload: &host.RpcMeta{
				PeerID: peerID,
				Messages: []host.RpcMetaMsg{
					{MsgID: "msg1", Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy"},
					{MsgID: "msg2", Topic: "/eth2/4a26c58b/beacon_attestation_0/ssz_snappy"},
					{MsgID: "msg3", Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy"}, // duplicate
				},
				Subscriptions: []host.RpcMetaSub{
					{TopicID: "/eth2/4a26c58b/voluntary_exit/ssz_snappy", Subscribe: true},
					{TopicID: "/eth2/4a26c58b/proposer_slashing/ssz_snappy", Subscribe: false},
				},
				Control: &host.RpcMetaControl{
					IHave: []host.RpcControlIHave{
						{TopicID: "/eth2/4a26c58b/attester_slashing/ssz_snappy", MsgIDs: []string{"msg4", "msg5"}},
					},
					Graft: []host.RpcControlGraft{
						{TopicID: "/eth2/4a26c58b/bls_to_execution_change/ssz_snappy"},
					},
					Prune: []host.RpcControlPrune{
						{TopicID: "/eth2/4a26c58b/sync_committee_contribution_and_proof/ssz_snappy"},
					},
				},
			},
		}

		topics := clmimicry.GetGossipTopics(event)

		// Should return unique topics from all sources
		expectedTopics := map[string]bool{
			"/eth2/4a26c58b/beacon_block/ssz_snappy":                          true,
			"/eth2/4a26c58b/beacon_attestation_0/ssz_snappy":                  true,
			"/eth2/4a26c58b/voluntary_exit/ssz_snappy":                        true,
			"/eth2/4a26c58b/proposer_slashing/ssz_snappy":                     true,
			"/eth2/4a26c58b/attester_slashing/ssz_snappy":                     true,
			"/eth2/4a26c58b/bls_to_execution_change/ssz_snappy":               true,
			"/eth2/4a26c58b/sync_committee_contribution_and_proof/ssz_snappy": true,
		}

		assert.Len(t, topics, len(expectedTopics))
		for _, topic := range topics {
			assert.True(t, expectedTopics[topic], "unexpected topic: %s", topic)
		}
	})

	t.Run("from map payload", func(t *testing.T) {
		tests := []struct {
			name     string
			payload  map[string]any
			expected []string
		}{
			{
				name: "single topic",
				payload: map[string]any{
					"Topic": "/eth2/4a26c58b/beacon_block/ssz_snappy",
					"MsgID": "msg-123",
				},
				expected: []string{"/eth2/4a26c58b/beacon_block/ssz_snappy"},
			},
			{
				name:     "no topic field",
				payload:  map[string]any{"MsgID": "msg-123"},
				expected: []string{},
			},
			{
				name:     "empty topic string",
				payload:  map[string]any{"Topic": ""},
				expected: []string{},
			},
			{
				name:     "topic not string",
				payload:  map[string]any{"Topic": 123},
				expected: []string{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				event := &host.TraceEvent{Payload: tt.payload}
				topics := clmimicry.GetGossipTopics(event)
				assert.Equal(t, tt.expected, topics)
			})
		}
	})

	t.Run("from struct with Topic field via reflection", func(t *testing.T) {
		// Test struct with plain Topic field
		type PayloadWithTopic struct {
			Topic string
			MsgID string
		}

		event := &host.TraceEvent{
			Payload: &PayloadWithTopic{
				Topic: "/eth2/test/topic",
				MsgID: "msg-123",
			},
		}

		topics := clmimicry.GetGossipTopics(event)
		// The reflection code doesn't extract plain string Topic fields
		assert.Equal(t, []string{}, topics)
	})

	t.Run("from struct with TopicId field via reflection", func(t *testing.T) {
		// Test struct with wrapperspb.StringValue TopicId field (like protobuf structs)
		type PayloadWithTopicId struct {
			TopicId *wrapperspb.StringValue
			MsgID   string
		}

		event := &host.TraceEvent{
			Payload: &PayloadWithTopicId{
				TopicId: wrapperspb.String("/eth2/test/topic_id"),
				MsgID:   "msg-123",
			},
		}

		topics := clmimicry.GetGossipTopics(event)
		assert.Equal(t, []string{"/eth2/test/topic_id"}, topics)
	})

	t.Run("edge cases", func(t *testing.T) {
		tests := []struct {
			name     string
			event    *host.TraceEvent
			expected []string
		}{
			{
				name:     "nil event",
				event:    nil,
				expected: nil,
			},
			{
				name:     "nil payload",
				event:    &host.TraceEvent{Payload: nil},
				expected: []string{},
			},
			{
				name:     "empty RpcMeta",
				event:    &host.TraceEvent{Payload: &host.RpcMeta{}},
				expected: []string{},
			},
			{
				name: "gossipsub event with topic in event",
				event: &host.TraceEvent{
					Topic:   "/eth2/12345678/beacon_attestation_1/ssz_snappy",
					Payload: map[string]any{"MsgID": "msg-123"},
				},
				expected: []string{"/eth2/12345678/beacon_attestation_1/ssz_snappy"},
			},
			{
				name: "RpcMeta with nil control",
				event: &host.TraceEvent{
					Payload: &host.RpcMeta{
						Messages: []host.RpcMetaMsg{
							{Topic: "/eth2/test/topic"},
						},
						Control: nil,
					},
				},
				expected: []string{"/eth2/test/topic"},
			},
			{
				name: "RpcMeta with empty topics",
				event: &host.TraceEvent{
					Payload: &host.RpcMeta{
						Messages: []host.RpcMetaMsg{
							{Topic: ""}, // empty topic
						},
						Subscriptions: []host.RpcMetaSub{
							{TopicID: ""}, // empty topic
						},
					},
				},
				expected: []string{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				topics := clmimicry.GetGossipTopics(tt.event)
				assert.Equal(t, tt.expected, topics)
			})
		}
	})

	t.Run("reflection safety - no panic on various types", func(t *testing.T) {
		// Test that reflection code doesn't panic on various field types
		type TestStruct struct {
			Topic      string
			TopicPtr   *string
			TopicId    *wrapperspb.StringValue
			TopicSlice []string
			TopicMap   map[string]string
			TopicChan  chan string
			TopicFunc  func() string
			TopicInt   int
		}

		topicStr := "/eth2/test/topic"
		event := &host.TraceEvent{
			Payload: &TestStruct{
				Topic:      topicStr,
				TopicPtr:   &topicStr,
				TopicId:    wrapperspb.String("/eth2/test/topic_id"),
				TopicSlice: []string{topicStr},
				TopicMap:   map[string]string{"key": topicStr},
				TopicChan:  nil, // nil channel
				TopicFunc:  nil, // nil func
				TopicInt:   123,
			},
		}

		// Should not panic
		require.NotPanics(t, func() {
			topics := clmimicry.GetGossipTopics(event)
			// Should extract only the TopicId field (wrapperspb.StringValue)
			// The plain string Topic field is not extracted by the reflection code
			assert.Equal(t, []string{"/eth2/test/topic_id"}, topics)
		})
	})
}

// Benchmark tests
func BenchmarkGetMsgID(b *testing.B) {
	b.Run("map payload", func(b *testing.B) {
		event := &host.TraceEvent{
			Payload: map[string]any{
				"MsgID": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"Topic": "/eth2/4a26c58b/beacon_block/ssz_snappy",
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = clmimicry.GetMsgID(event)
		}
	})

	b.Run("struct payload", func(b *testing.B) {
		type TestPayload struct {
			MsgID string
			Topic string
		}

		event := &host.TraceEvent{
			Payload: &TestPayload{
				MsgID: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy",
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = clmimicry.GetMsgID(event)
		}
	})
}

func BenchmarkGetGossipTopics_Large(b *testing.B) {
	// Benchmark with a large RpcMeta payload
	peerID, _ := peer.Decode("16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq")

	messages := make([]host.RpcMetaMsg, 100)
	for i := range messages {
		messages[i] = host.RpcMetaMsg{
			MsgID: "msg-" + string(rune(i)),
			Topic: "/eth2/4a26c58b/topic_" + string(rune(i%10)) + "/ssz_snappy",
		}
	}

	event := &host.TraceEvent{
		Type:   "LIBP2P_TRACE_RECV_RPC",
		PeerID: peerID,
		Payload: &host.RpcMeta{
			PeerID:   peerID,
			Messages: messages,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = clmimicry.GetGossipTopics(event)
	}
}
