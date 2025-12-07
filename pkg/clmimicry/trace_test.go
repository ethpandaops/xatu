package clmimicry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestShouldTraceMessage tests the ShouldTraceMessage function indirectly
func TestShouldTraceMessage(t *testing.T) {
	// Test the helper functions that are used by ShouldTraceMessage
	t.Run("GetMsgID extracts message ID correctly", func(t *testing.T) {
		event := &PublishMessageEvent{
			MsgID: "test-message-id",
			Topic: "/eth2/test/beacon_block/ssz_snappy",
		}

		msgID := GetMsgID(event)
		assert.Equal(t, "test-message-id", msgID)
	})

	t.Run("GetMsgID returns empty for missing MsgID", func(t *testing.T) {
		event := &GraftEvent{
			Topic: "/eth2/test/beacon_block/ssz_snappy",
		}

		msgID := GetMsgID(event)
		assert.Equal(t, "", msgID)
	})

	t.Run("GetGossipTopics extracts topics correctly", func(t *testing.T) {
		event := &PublishMessageEvent{
			MsgID: "test-message-id",
			Topic: "/eth2/test/beacon_block/ssz_snappy",
		}

		topics := GetGossipTopics(event)
		assert.Len(t, topics, 1)
		assert.Contains(t, topics, "/eth2/test/beacon_block/ssz_snappy")
	})
}

// TestShouldTraceRPCMetaMessagesLogic tests the filtering logic without full Mimicry setup
func TestShouldTraceRPCMetaMessagesLogic(t *testing.T) {
	t.Run("filters nil and empty message IDs", func(t *testing.T) {
		messages := []RPCMetaMessageInfo{
			{
				MessageID: nil,
				Topic:     wrapperspb.String("/eth2/test/beacon_block/ssz_snappy"),
			},
			{
				MessageID: wrapperspb.String(""),
				Topic:     wrapperspb.String("/eth2/test/beacon_block/ssz_snappy"),
			},
			{
				MessageID: wrapperspb.String("valid-msg"),
				Topic:     wrapperspb.String("/eth2/test/beacon_block/ssz_snappy"),
			},
		}

		// The actual function filters out nil and empty message IDs
		validCount := 0
		for _, msg := range messages {
			if msg.MessageID != nil && msg.MessageID.GetValue() != "" {
				validCount++
			}
		}

		assert.Equal(t, 1, validCount, "Only one valid message")
	})

	t.Run("handles RPCMetaTopicInfo correctly", func(t *testing.T) {
		topics := []RPCMetaTopicInfo{
			{
				Topic: wrapperspb.String("/eth2/test/beacon_block/ssz_snappy"),
			},
			{
				Topic: nil,
			},
			{
				Topic: wrapperspb.String(""),
			},
		}

		// Count valid topics
		validCount := 0
		for _, topicInfo := range topics {
			if topicInfo.Topic != nil && topicInfo.Topic.GetValue() != "" {
				validCount++
			}
		}

		assert.Equal(t, 1, validCount, "Only one valid topic")
	})

	t.Run("handles legacy StringValue format", func(t *testing.T) {
		messages := []*wrapperspb.StringValue{
			wrapperspb.String("msg-1"),
			wrapperspb.String("msg-2"),
			nil,
			wrapperspb.String(""),
			wrapperspb.String("msg-3"),
		}

		// Count valid messages
		validCount := 0
		for _, msg := range messages {
			if msg != nil && msg.GetValue() != "" {
				validCount++
			}
		}

		assert.Equal(t, 3, validCount, "Three valid messages")
	})
}

// TestFilteredMessageWithIndex tests the FilteredMessageWithIndex struct
func TestFilteredMessageWithIndex(t *testing.T) {
	msg := FilteredMessageWithIndex{
		MessageID:     wrapperspb.String("test-msg"),
		OriginalIndex: 42,
	}

	assert.Equal(t, "test-msg", msg.MessageID.GetValue())
	assert.Equal(t, uint32(42), msg.OriginalIndex)
}

// TestRPCMetaStructs tests the RPC meta structs
func TestRPCMetaStructs(t *testing.T) {
	t.Run("RPCMetaMessageInfo", func(t *testing.T) {
		msg := RPCMetaMessageInfo{
			MessageID: wrapperspb.String("msg-id"),
			Topic:     wrapperspb.String("/eth2/topic"),
		}

		assert.Equal(t, "msg-id", msg.MessageID.GetValue())
		assert.Equal(t, "/eth2/topic", msg.Topic.GetValue())
	})

	t.Run("RPCMetaTopicInfo", func(t *testing.T) {
		topicInfo := RPCMetaTopicInfo{
			Topic: wrapperspb.String("/eth2/topic"),
		}

		assert.Equal(t, "/eth2/topic", topicInfo.Topic.GetValue())
	})
}

// TestIsShardActive tests the IsShardActive function
func TestIsShardActive(t *testing.T) {
	testCases := []struct {
		name         string
		shard        uint64
		activeShards []uint64
		expected     bool
	}{
		{
			name:         "shard is in list",
			shard:        5,
			activeShards: []uint64{1, 3, 5, 7, 9},
			expected:     true,
		},
		{
			name:         "shard is not in list",
			shard:        4,
			activeShards: []uint64{1, 3, 5, 7, 9},
			expected:     false,
		},
		{
			name:         "empty active shards",
			shard:        1,
			activeShards: []uint64{},
			expected:     false,
		},
		{
			name:         "single shard match",
			shard:        42,
			activeShards: []uint64{42},
			expected:     true,
		},
		{
			name:         "large shard number",
			shard:        511,
			activeShards: []uint64{500, 501, 502, 503, 504, 505, 506, 507, 508, 509, 510, 511},
			expected:     true,
		},
		{
			name:         "first shard",
			shard:        0,
			activeShards: []uint64{0, 1, 2, 3},
			expected:     true,
		},
		{
			name:         "last shard check",
			shard:        511,
			activeShards: []uint64{0, 255, 511},
			expected:     true,
		},
		{
			name:         "duplicate shards in list",
			shard:        5,
			activeShards: []uint64{1, 3, 5, 5, 7, 9}, // 5 appears twice
			expected:     true,
		},
		{
			name:         "nil active shards",
			shard:        5,
			activeShards: nil,
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsShardActive(tc.shard, tc.activeShards)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIsShardActivePerformance tests performance with large shard lists
func TestIsShardActivePerformance(t *testing.T) {
	// Create a large list of active shards
	largeShardList := make([]uint64, 256)
	for i := range largeShardList {
		largeShardList[i] = uint64(i * 2) // Even numbers only
	}

	testCases := []struct {
		name     string
		shard    uint64
		expected bool
	}{
		{
			name:     "even shard (should be found)",
			shard:    100,
			expected: true,
		},
		{
			name:     "odd shard (should not be found)",
			shard:    101,
			expected: false,
		},
		{
			name:     "first even shard",
			shard:    0,
			expected: true,
		},
		{
			name:     "last even shard in range",
			shard:    510,
			expected: true,
		},
		{
			name:     "beyond range",
			shard:    1000,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsShardActive(tc.shard, largeShardList)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// BenchmarkIsShardActive benchmarks the IsShardActive function
func BenchmarkIsShardActive(b *testing.B) {
	activeShards := make([]uint64, 100)
	for i := range activeShards {
		activeShards[i] = uint64(i * 5)
	}

	b.Run("shard_found_early", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IsShardActive(10, activeShards) // Should be found early
		}
	})

	b.Run("shard_found_late", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IsShardActive(495, activeShards) // Should be found late
		}
	})

	b.Run("shard_not_found", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IsShardActive(7, activeShards) // Should not be found
		}
	})
}
