package clmimicry

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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

// RPCMetaTopicInfo represents a topic-based RPC meta event for filtering
type RPCMetaTopicInfo struct {
	Topic *wrapperspb.StringValue // Gossip topic for the event
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
func (p *Processor) ShouldTraceMessage(
	event TraceEvent,
	clientMeta *xatu.ClientMeta,
	xatuEventType string,
) bool {
	if p.unifiedSharder == nil || !p.unifiedSharder.enabled {
		// If sharding is disabled, process all messages
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddProcessedMessage(xatuEventType, networkStr)

		return true
	}

	var (
		// Get the event type from the string.
		eventType = xatu.Event_Name(xatu.Event_Name_value[xatuEventType])

		// Extract message ID and topics using the existing helper functions.
		msgID = GetMsgID(event)

		// Get topic from the event if it implements TopicEvent.
		topic string
	)

	if te, ok := event.(TopicEvent); ok {
		topic = te.GetTopic()
	}

	// Use the unified sharder to determine if we should process this event
	shouldProcess, reason := p.unifiedSharder.ShouldProcess(eventType, msgID, topic)

	// Update metrics
	networkStr := getNetworkID(clientMeta)
	p.metrics.AddShardingDecision(xatuEventType, reason, networkStr)

	if shouldProcess {
		p.metrics.AddProcessedMessage(xatuEventType, networkStr)
	} else {
		p.metrics.AddSkippedMessage(xatuEventType, networkStr)
	}

	return shouldProcess
}

// ShouldTraceRPCMetaMessages determines which RPC meta messages should be processed based on sharding configuration
func (p *Processor) ShouldTraceRPCMetaMessages(
	clientMeta *xatu.ClientMeta,
	xatuEventType string,
	messages interface{}, // []RPCMetaMessageInfo, []RPCMetaTopicInfo, or []*wrapperspb.StringValue
) ([]FilteredMessageWithIndex, error) {
	if p.unifiedSharder == nil || !p.unifiedSharder.enabled {
		// If sharding is disabled, include all messages
		return p.buildAllMessagesFromInterface(messages, xatuEventType, getNetworkID(clientMeta)), nil
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

			shouldProcess, reason := p.unifiedSharder.ShouldProcess(eventType, msg.MessageID.GetValue(), topic)
			p.metrics.AddShardingDecision(xatuEventType, reason, networkStr)

			if shouldProcess {
				filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
					MessageID:     msg.MessageID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				p.metrics.AddProcessedMessage(xatuEventType, networkStr)
			} else {
				p.metrics.AddSkippedMessage(xatuEventType, networkStr)
			}
		}

	case []RPCMetaTopicInfo:
		for i, topicInfo := range msgs {
			topic := ""
			if topicInfo.Topic != nil {
				topic = topicInfo.Topic.GetValue()
			}

			// For Group B events, sharding is based on topic only. We pass empty string
			// for msgID since these events don't have message IDs.
			shouldProcess, reason := p.unifiedSharder.ShouldProcess(eventType, "", topic)
			p.metrics.AddShardingDecision(xatuEventType, reason, networkStr)

			if shouldProcess {
				filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				p.metrics.AddProcessedMessage(xatuEventType, networkStr)
			} else {
				p.metrics.AddSkippedMessage(xatuEventType, networkStr)
			}
		}

	case []*wrapperspb.StringValue:
		for i, msgID := range msgs {
			if msgID == nil || msgID.GetValue() == "" {
				continue
			}

			// No topic information available for legacy format
			shouldProcess, reason := p.unifiedSharder.ShouldProcess(eventType, msgID.GetValue(), "")
			p.metrics.AddShardingDecision(xatuEventType, reason, networkStr)

			if shouldProcess {
				filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
					MessageID:     msgID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				p.metrics.AddProcessedMessage(xatuEventType, networkStr)
			} else {
				p.metrics.AddSkippedMessage(xatuEventType, networkStr)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported message type: %T", messages)
	}

	return filteredMessages, nil
}

// buildAllMessagesFromInterface builds a result with all messages when sharding is disabled
func (p *Processor) buildAllMessagesFromInterface(messages interface{}, xatuEventType, networkStr string) []FilteredMessageWithIndex {
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

				p.metrics.AddProcessedMessage(xatuEventType, networkStr)
			}
		}
	case []RPCMetaTopicInfo:
		result = make([]FilteredMessageWithIndex, 0, len(msgs))

		for i := range msgs {
			result = append(result, FilteredMessageWithIndex{
				OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
			})

			p.metrics.AddProcessedMessage(xatuEventType, networkStr)
		}
	case []*wrapperspb.StringValue:
		result = make([]FilteredMessageWithIndex, 0, len(msgs))

		for i, msgID := range msgs {
			if msgID != nil {
				result = append(result, FilteredMessageWithIndex{
					MessageID:     msgID,
					OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
				})

				p.metrics.AddProcessedMessage(xatuEventType, networkStr)
			}
		}
	}

	return result
}
