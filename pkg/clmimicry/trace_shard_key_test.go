package clmimicry_test

import (
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/clmimicry"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMsgID(t *testing.T) {
	peerID, err := peer.Decode("16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq")
	require.NoError(t, err)

	t.Run("from PublishMessageEvent", func(t *testing.T) {
		event := &clmimicry.PublishMessageEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			MsgID: "msg-123",
			Topic: "/eth2/test/beacon_block/ssz_snappy",
		}

		result := clmimicry.GetMsgID(event)
		assert.Equal(t, "msg-123", result)
	})

	t.Run("from DeliverMessageEvent", func(t *testing.T) {
		event := &clmimicry.DeliverMessageEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			MsgID: "deliver-msg-456",
			Topic: "/eth2/test/beacon_block/ssz_snappy",
		}

		result := clmimicry.GetMsgID(event)
		assert.Equal(t, "deliver-msg-456", result)
	})

	t.Run("from RejectMessageEvent", func(t *testing.T) {
		event := &clmimicry.RejectMessageEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			MsgID:  "reject-msg-789",
			Topic:  "/eth2/test/beacon_block/ssz_snappy",
			Reason: "validation_failed",
		}

		result := clmimicry.GetMsgID(event)
		assert.Equal(t, "reject-msg-789", result)
	})

	t.Run("from DuplicateMessageEvent", func(t *testing.T) {
		event := &clmimicry.DuplicateMessageEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			MsgID: "dup-msg-000",
			Topic: "/eth2/test/beacon_block/ssz_snappy",
		}

		result := clmimicry.GetMsgID(event)
		assert.Equal(t, "dup-msg-000", result)
	})

	t.Run("from non-message event (GraftEvent)", func(t *testing.T) {
		event := &clmimicry.GraftEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			Topic: "/eth2/test/beacon_block/ssz_snappy",
		}

		// GraftEvent doesn't implement MessageEvent, so should return empty string
		result := clmimicry.GetMsgID(event)
		assert.Equal(t, "", result)
	})

	t.Run("nil event", func(t *testing.T) {
		result := clmimicry.GetMsgID(nil)
		assert.Equal(t, "", result)
	})
}

func TestGetGossipTopics(t *testing.T) {
	peerID, err := peer.Decode("16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq")
	require.NoError(t, err)

	t.Run("from RPC event with RpcMeta", func(t *testing.T) {
		event := &clmimicry.RecvRPCEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			Meta: &clmimicry.RpcMeta{
				PeerID: peerID,
				Messages: []clmimicry.RpcMetaMsg{
					{MsgID: "msg1", Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy"},
					{MsgID: "msg2", Topic: "/eth2/4a26c58b/beacon_attestation_0/ssz_snappy"},
					{MsgID: "msg3", Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy"}, // duplicate
				},
				Subscriptions: []clmimicry.RpcMetaSub{
					{TopicID: "/eth2/4a26c58b/voluntary_exit/ssz_snappy", Subscribe: true},
					{TopicID: "/eth2/4a26c58b/proposer_slashing/ssz_snappy", Subscribe: false},
				},
				Control: &clmimicry.RpcMetaControl{
					IHave: []clmimicry.RpcControlIHave{
						{TopicID: "/eth2/4a26c58b/attester_slashing/ssz_snappy", MsgIDs: []string{"msg4", "msg5"}},
					},
					Graft: []clmimicry.RpcControlGraft{
						{TopicID: "/eth2/4a26c58b/bls_to_execution_change/ssz_snappy"},
					},
					Prune: []clmimicry.RpcControlPrune{
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

	t.Run("from TopicEvent (JoinEvent)", func(t *testing.T) {
		event := &clmimicry.JoinEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy",
		}

		topics := clmimicry.GetGossipTopics(event)
		assert.Equal(t, []string{"/eth2/4a26c58b/beacon_block/ssz_snappy"}, topics)
	})

	t.Run("from TopicEvent (LeaveEvent)", func(t *testing.T) {
		event := &clmimicry.LeaveEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			Topic: "/eth2/4a26c58b/beacon_attestation_5/ssz_snappy",
		}

		topics := clmimicry.GetGossipTopics(event)
		assert.Equal(t, []string{"/eth2/4a26c58b/beacon_attestation_5/ssz_snappy"}, topics)
	})

	t.Run("from TopicEvent (GraftEvent)", func(t *testing.T) {
		event := &clmimicry.GraftEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			Topic: "/eth2/4a26c58b/voluntary_exit/ssz_snappy",
		}

		topics := clmimicry.GetGossipTopics(event)
		assert.Equal(t, []string{"/eth2/4a26c58b/voluntary_exit/ssz_snappy"}, topics)
	})

	t.Run("from TopicEvent (PruneEvent)", func(t *testing.T) {
		event := &clmimicry.PruneEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			Topic: "/eth2/4a26c58b/proposer_slashing/ssz_snappy",
		}

		topics := clmimicry.GetGossipTopics(event)
		assert.Equal(t, []string{"/eth2/4a26c58b/proposer_slashing/ssz_snappy"}, topics)
	})

	t.Run("from TopicEvent (PublishMessageEvent)", func(t *testing.T) {
		event := &clmimicry.PublishMessageEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			MsgID: "msg-123",
			Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy",
		}

		topics := clmimicry.GetGossipTopics(event)
		assert.Equal(t, []string{"/eth2/4a26c58b/beacon_block/ssz_snappy"}, topics)
	})

	t.Run("edge cases", func(t *testing.T) {
		tests := []struct {
			name     string
			event    clmimicry.TraceEvent
			expected []string
		}{
			{
				name:     "nil event",
				event:    nil,
				expected: nil,
			},
			{
				name: "empty RpcMeta",
				event: &clmimicry.RecvRPCEvent{
					TraceEventBase: clmimicry.TraceEventBase{
						Timestamp: time.Now(),
						PeerID:    peerID,
					},
					Meta: &clmimicry.RpcMeta{},
				},
				expected: []string{},
			},
			{
				name: "RpcMeta with nil control",
				event: &clmimicry.RecvRPCEvent{
					TraceEventBase: clmimicry.TraceEventBase{
						Timestamp: time.Now(),
						PeerID:    peerID,
					},
					Meta: &clmimicry.RpcMeta{
						Messages: []clmimicry.RpcMetaMsg{
							{Topic: "/eth2/test/topic"},
						},
						Control: nil,
					},
				},
				expected: []string{"/eth2/test/topic"},
			},
			{
				name: "RpcMeta with empty topics",
				event: &clmimicry.RecvRPCEvent{
					TraceEventBase: clmimicry.TraceEventBase{
						Timestamp: time.Now(),
						PeerID:    peerID,
					},
					Meta: &clmimicry.RpcMeta{
						Messages: []clmimicry.RpcMetaMsg{
							{Topic: ""}, // empty topic
						},
						Subscriptions: []clmimicry.RpcMetaSub{
							{TopicID: ""}, // empty topic
						},
					},
				},
				expected: []string{},
			},
			{
				name: "event without Topic (AddPeerEvent)",
				event: &clmimicry.AddPeerEvent{
					TraceEventBase: clmimicry.TraceEventBase{
						Timestamp: time.Now(),
						PeerID:    peerID,
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
}

// Benchmark tests
func BenchmarkGetMsgID(b *testing.B) {
	peerID, _ := peer.Decode("16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq")

	b.Run("PublishMessageEvent", func(b *testing.B) {
		event := &clmimicry.PublishMessageEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			MsgID: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = clmimicry.GetMsgID(event)
		}
	})

	b.Run("DeliverMessageEvent", func(b *testing.B) {
		event := &clmimicry.DeliverMessageEvent{
			TraceEventBase: clmimicry.TraceEventBase{
				Timestamp: time.Now(),
				PeerID:    peerID,
			},
			MsgID: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Topic: "/eth2/4a26c58b/beacon_block/ssz_snappy",
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

	messages := make([]clmimicry.RpcMetaMsg, 100)
	for i := range messages {
		messages[i] = clmimicry.RpcMetaMsg{
			MsgID: "msg-" + string(rune(i)),
			Topic: "/eth2/4a26c58b/topic_" + string(rune(i%10)) + "/ssz_snappy",
		}
	}

	event := &clmimicry.RecvRPCEvent{
		TraceEventBase: clmimicry.TraceEventBase{
			Timestamp: time.Now(),
			PeerID:    peerID,
		},
		Meta: &clmimicry.RpcMeta{
			PeerID:   peerID,
			Messages: messages,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = clmimicry.GetGossipTopics(event)
	}
}
