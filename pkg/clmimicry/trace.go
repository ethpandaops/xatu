package clmimicry

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// FilteredMessageWithIndex represents a filtered message with its original index
type FilteredMessageWithIndex struct {
	MessageID     *wrapperspb.StringValue
	OriginalIndex uint32
}

// RPCMetaMessageInfo represents a message with its ID and optional topic for RPC meta filtering
type RPCMetaMessageInfo struct {
	MessageID *wrapperspb.StringValue
	Topic     *wrapperspb.StringValue // Optional: gossip topic for the message
}

// RPCMetaPeerInfo represents a peer with its ID and optional topic for RPC meta filtering
type RPCMetaPeerInfo struct {
	PeerID *wrapperspb.StringValue
	Topic  *wrapperspb.StringValue // Optional: gossip topic for the peer action
}

// IsShardActive checks if a shard is in the active shards list.
func IsShardActive(shard uint64, activeShards []uint64) bool {
	for _, activeShard := range activeShards {
		if shard == activeShard {
			return true
		}
	}

	return false
}

// ShouldTraceMessage determines whether a message with the given MsgID should be included
// in the sample based on the configured trace settings.
func (m *Mimicry) ShouldTraceMessage(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	xatuEventType string,
) bool {
	if m.sharder == nil || !m.sharder.enabled {
		// If sharding is disabled, process all messages
		networkStr := getNetworkID(clientMeta)
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)

		return true
	}

	// Get the event type from the string
	eventType := xatu.Event_Name(xatu.Event_Name_value[xatuEventType])

	// Extract message ID and topics using the existing helper functions
	msgID := GetMsgID(event)

	// Get all topics from the event
	topics := GetGossipTopics(event)
	topic := ""

	if len(topics) > 0 {
		topic = topics[0] // Use the first topic if available
	}

	// Use the unified sharder to determine if we should process this event
	shouldProcess, reason := m.sharder.ShouldProcess(eventType, msgID, topic)

	// Update metrics
	networkStr := getNetworkID(clientMeta)
	m.metrics.AddShardingDecision(xatuEventType, reason, networkStr)

	if shouldProcess {
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)
	} else {
		m.metrics.AddSkippedMessage(xatuEventType, networkStr)
	}

	return shouldProcess
}

// ShouldTraceRPCMetaMessages filters RPC meta messages based on sharding configuration.
func (m *Mimicry) ShouldTraceRPCMetaMessages(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	xatuEventType string,
	messages interface{}, // []RPCMetaMessageInfo, []RPCMetaPeerInfo, or []*wrapperspb.StringValue
) ([]FilteredMessageWithIndex, error) {
	if m.sharder == nil || !m.sharder.enabled {
		// If sharding is disabled, include all messages
		return m.buildAllMessagesFromInterface(messages, xatuEventType, getNetworkID(clientMeta)), nil
	}

	// Get the event type from the string
	eventType := xatu.Event_Name(xatu.Event_Name_value[xatuEventType])

	var filteredMessages []FilteredMessageWithIndex
	networkStr := getNetworkID(clientMeta)

	// Process based on message type
	switch msgs := messages.(type) {
	case []RPCMetaMessageInfo:
		for i, msg := range msgs {
			if msg.MessageID == nil || msg.MessageID.GetValue() == "" {
				continue
			}

			topic := ""
			if msg.Topic != nil {
				topic = msg.Topic.GetValue()
			}

			shouldProcess, reason := m.sharder.ShouldProcess(eventType, msg.MessageID.GetValue(), topic)
			m.metrics.AddShardingDecision(xatuEventType, reason, networkStr)

			if shouldProcess {
				filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
					MessageID:     msg.MessageID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				m.metrics.AddProcessedMessage(xatuEventType, networkStr)
			} else {
				m.metrics.AddSkippedMessage(xatuEventType, networkStr)
			}
		}

	case []RPCMetaPeerInfo:
		for i, peer := range msgs {
			if peer.PeerID == nil || peer.PeerID.GetValue() == "" {
				continue
			}

			topic := ""
			if peer.Topic != nil {
				topic = peer.Topic.GetValue()
			}

			// For Group B events (PRUNE), sharding is based on topic only
			// We pass empty string for msgID since these events don't have message IDs
			shouldProcess, reason := m.sharder.ShouldProcess(eventType, "", topic)
			m.metrics.AddShardingDecision(xatuEventType, reason, networkStr)

			if shouldProcess {
				filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
					MessageID:     peer.PeerID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				m.metrics.AddProcessedMessage(xatuEventType, networkStr)
			} else {
				m.metrics.AddSkippedMessage(xatuEventType, networkStr)
			}
		}

	case []*wrapperspb.StringValue:
		for i, msgID := range msgs {
			if msgID == nil || msgID.GetValue() == "" {
				continue
			}

			// No topic information available for legacy format
			shouldProcess, reason := m.sharder.ShouldProcess(eventType, msgID.GetValue(), "")
			m.metrics.AddShardingDecision(xatuEventType, reason, networkStr)

			if shouldProcess {
				filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
					MessageID:     msgID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				m.metrics.AddProcessedMessage(xatuEventType, networkStr)
			} else {
				m.metrics.AddSkippedMessage(xatuEventType, networkStr)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported message type: %T", messages)
	}

	return filteredMessages, nil
}

// buildAllMessagesFromInterface builds a result with all messages when sharding is disabled
func (m *Mimicry) buildAllMessagesFromInterface(messages interface{}, xatuEventType, networkStr string) []FilteredMessageWithIndex {
	var result []FilteredMessageWithIndex

	switch msgs := messages.(type) {
	case []RPCMetaMessageInfo:
		result = make([]FilteredMessageWithIndex, 0, len(msgs))

		for i, msg := range msgs {
			if msg.MessageID != nil {
				result = append(result, FilteredMessageWithIndex{
					MessageID:     msg.MessageID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				m.metrics.AddProcessedMessage(xatuEventType, networkStr)
			}
		}
	case []RPCMetaPeerInfo:
		result = make([]FilteredMessageWithIndex, 0, len(msgs))

		for i, peer := range msgs {
			if peer.PeerID != nil {
				result = append(result, FilteredMessageWithIndex{
					MessageID:     peer.PeerID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				m.metrics.AddProcessedMessage(xatuEventType, networkStr)
			}
		}
	case []*wrapperspb.StringValue:
		result = make([]FilteredMessageWithIndex, 0, len(msgs))

		for i, msgID := range msgs {
			if msgID != nil {
				result = append(result, FilteredMessageWithIndex{
					MessageID:     msgID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				m.metrics.AddProcessedMessage(xatuEventType, networkStr)
			}
		}
	}

	return result
}
