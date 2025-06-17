package clmimicry

import (
	"fmt"
	"regexp"

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
	networkStr := getNetworkID(clientMeta)

	// If the event type is unshardable, we can move on with life.
	if isUnshardableEvent(xatuEventType) {
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)

		return true
	}

	// Check if there's a matching topic config in the trace-based configuration.
	if m.Config.Traces.Enabled {
		topicConfig, found := m.Config.Traces.FindMatchingTopicConfig(xatuEventType)
		if found {
			// Check if this is hierarchical or simple configuration
			if topicConfig.Topics != nil {
				// Use hierarchical sharding logic
				return m.shouldTraceWithHierarchicalConfig(event, clientMeta, topicConfig, xatuEventType)
			}

			// Use simple sharding logic (backward compatible)
			return m.shouldTraceWithSimpleConfig(event, clientMeta, topicConfig, xatuEventType)
		}
	}

	// If no trace-based config matched, process all messages for enabled event types.
	m.metrics.AddProcessedMessage(xatuEventType, networkStr)

	return true
}

// ShouldTraceRPCMetaMessages filters RPC meta messages with support for both simple and hierarchical configurations.
// It accepts either []RPCMetaMessageInfo (with topic support) or []*wrapperspb.StringValue (legacy format).
func (m *Mimicry) ShouldTraceRPCMetaMessages(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	xatuEventType string,
	messages interface{}, // []RPCMetaMessageInfo or []*wrapperspb.StringValue
) ([]FilteredMessageWithIndex, error) {
	// Delegate to appropriate method based on message type
	switch msgs := messages.(type) {
	case []RPCMetaMessageInfo:
		return m.shouldTraceRPCMetaMessagesWithTopics(event, clientMeta, xatuEventType, msgs)
	case []*wrapperspb.StringValue:
		return m.shouldTraceRPCMetaMessagesWithMessageIDs(event, clientMeta, xatuEventType, msgs)
	default:
		return nil, fmt.Errorf("unsupported message type: %T", messages)
	}
}

// shouldTraceRPCMetaMessagesWithTopics handles topic-aware RPC meta message filtering
func (m *Mimicry) shouldTraceRPCMetaMessagesWithTopics(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	xatuEventType string,
	messages []RPCMetaMessageInfo,
) ([]FilteredMessageWithIndex, error) {
	networkStr := getNetworkID(clientMeta)

	// Check early exit conditions
	if shouldSkipFiltering, result := m.shouldSkipRPCFiltering(xatuEventType, networkStr, len(messages), func(i int) *wrapperspb.StringValue {
		return messages[i].MessageID
	}); shouldSkipFiltering {
		return result, nil
	}

	topicConfig, found := m.Config.Traces.FindMatchingTopicConfig(xatuEventType)
	if !found {
		return m.buildAllMessagesResult(len(messages), func(i int) *wrapperspb.StringValue {
			return messages[i].MessageID
		}, xatuEventType, networkStr), nil
	}

	// Check if this is hierarchical or simple configuration
	if topicConfig.Topics != nil {
		// Hierarchical configuration: filter each message based on its gossip topic
		return m.filterRPCMetaMessagesHierarchical(messages, topicConfig, xatuEventType, networkStr)
	}

	// Simple configuration logic (backward compatible)
	if topicConfig.TotalShards == nil {
		return m.buildAllMessagesResult(len(messages), func(i int) *wrapperspb.StringValue {
			return messages[i].MessageID
		}, xatuEventType, networkStr), nil
	}

	// Convert messages to message IDs for simple configuration
	messageIDs := make([]*wrapperspb.StringValue, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.MessageID
	}

	return m.filterRPCMetaMessagesWithSimpleConfig(messageIDs, topicConfig, xatuEventType, networkStr)
}

// shouldTraceRPCMetaMessagesWithMessageIDs handles RPC meta message filtering using message IDs only
func (m *Mimicry) shouldTraceRPCMetaMessagesWithMessageIDs(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	xatuEventType string,
	messageIDs []*wrapperspb.StringValue,
) ([]FilteredMessageWithIndex, error) {
	networkStr := getNetworkID(clientMeta)

	// Check early exit conditions
	if shouldSkipFiltering, result := m.shouldSkipRPCFiltering(xatuEventType, networkStr, len(messageIDs), func(i int) *wrapperspb.StringValue {
		return messageIDs[i]
	}); shouldSkipFiltering {
		return result, nil
	}

	topicConfig, found := m.Config.Traces.FindMatchingTopicConfig(xatuEventType)
	if !found {
		return m.buildAllMessagesResult(len(messageIDs), func(i int) *wrapperspb.StringValue {
			return messageIDs[i]
		}, xatuEventType, networkStr), nil
	}

	// Check if this is hierarchical or simple configuration
	if topicConfig.Topics != nil {
		// For hierarchical configuration without topic information, use fallback configuration.
		// Note: For RPC meta messages with topic information, use ShouldTraceRPCMetaMessagesWithTopics instead.
		gossipConfig, configFound := m.getGossipTopicConfig(topicConfig, "")
		if !configFound {
			// No fallback config, return empty
			return []FilteredMessageWithIndex{}, nil
		}

		return m.filterRPCMetaMessagesWithConfig(messageIDs, gossipConfig, xatuEventType, networkStr)
	}

	// Simple configuration logic (backward compatible)
	if topicConfig.TotalShards == nil {
		return m.buildAllMessagesResult(len(messageIDs), func(i int) *wrapperspb.StringValue {
			return messageIDs[i]
		}, xatuEventType, networkStr), nil
	}

	return m.filterRPCMetaMessagesWithSimpleConfig(messageIDs, topicConfig, xatuEventType, networkStr)
}

// shouldTraceWithHierarchicalConfig determines if an event should be traced using hierarchical configuration.
func (m *Mimicry) shouldTraceWithHierarchicalConfig(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	topicConfig *TopicConfig,
	xatuEventType string,
) bool {
	networkStr := getNetworkID(clientMeta)

	// Extract gossip topic from the event
	gossipTopic := GetGossipTopic(event)

	// Get the appropriate gossip topic configuration
	gossipConfig, found := m.getGossipTopicConfig(topicConfig, gossipTopic)
	if !found {
		// No configuration found, skip this message
		m.metrics.AddSkippedMessage(xatuEventType, networkStr)

		return false
	}

	// Get the appropriate sharding key based on the gossip configuration
	shardingKey := GetShardingKey(event, clientMeta, gossipConfig.ShardingKey, xatuEventType)

	// If no sharding key, we can't sample. Shouldn't ever happen.
	if shardingKey == "" {
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)

		return true
	}

	// If all shards are configured to be active, skip hashing and return true, save some trees.
	//nolint:gosec // controlled config, no overflow.
	if len(gossipConfig.ActiveShards) == int(gossipConfig.TotalShards) {
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)

		return true
	}

	// Calculate the shard for this message.
	shard := GetShard(shardingKey, gossipConfig.TotalShards)

	// Record metrics for all messages to track distribution.
	m.metrics.AddShardObservation(xatuEventType, shard, networkStr)

	// Check if this shard is in the active shards list.
	isActive := IsShardActive(shard, gossipConfig.ActiveShards)

	// Record processed or skipped metrics.
	if isActive {
		m.metrics.AddShardProcessed(xatuEventType, shard, networkStr)
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)
	} else {
		m.metrics.AddShardSkipped(xatuEventType, shard, networkStr)
		m.metrics.AddSkippedMessage(xatuEventType, networkStr)
	}

	return isActive
}

// shouldTraceWithSimpleConfig determines if an event should be traced using simple configuration.
func (m *Mimicry) shouldTraceWithSimpleConfig(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	topicConfig *TopicConfig,
	xatuEventType string,
) bool {
	networkStr := getNetworkID(clientMeta)

	// Simple configuration logic (backward compatible)
	if topicConfig.TotalShards == nil {
		// Invalid simple config, should not happen after validation
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)

		return true
	}

	totalShards := *topicConfig.TotalShards

	// Get the appropriate sharding key based on the configuration
	shardingKey := GetShardingKey(event, clientMeta, topicConfig.ShardingKey, xatuEventType)

	// If no sharding key, we can't sample. Shouldn't ever happen.
	if shardingKey == "" {
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)

		return true
	}

	// If all shards are configured to be active, skip hashing and return true, save some trees.
	//nolint:gosec // controlled config, no overflow.
	if len(topicConfig.ActiveShards) == int(totalShards) {
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)

		return true
	}

	// Calculate the shard for this message.
	shard := GetShard(shardingKey, totalShards)

	// Record metrics for all messages to track distribution.
	m.metrics.AddShardObservation(xatuEventType, shard, networkStr)

	// Check if this shard is in the active shards list.
	isActive := IsShardActive(shard, topicConfig.ActiveShards)

	// Record processed or skipped metrics.
	if isActive {
		m.metrics.AddShardProcessed(xatuEventType, shard, networkStr)
		m.metrics.AddProcessedMessage(xatuEventType, networkStr)
	} else {
		m.metrics.AddShardSkipped(xatuEventType, shard, networkStr)
		m.metrics.AddSkippedMessage(xatuEventType, networkStr)
	}

	return isActive
}

// filterRPCMetaMessagesWithConfig filters RPC meta messages using a specific gossip topic configuration.
func (m *Mimicry) filterRPCMetaMessagesWithConfig(
	messageIDs []*wrapperspb.StringValue,
	gossipConfig *GossipTopicConfig,
	xatuEventType string,
	networkStr string,
) ([]FilteredMessageWithIndex, error) {
	// If all shards are configured to be active, skip filtering
	//nolint:gosec // controlled config, no overflow.
	if len(gossipConfig.ActiveShards) == int(gossipConfig.TotalShards) {
		result := make([]FilteredMessageWithIndex, len(messageIDs))

		for i, messageID := range messageIDs {
			result[i] = FilteredMessageWithIndex{
				MessageID:     messageID,
				OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
			}

			m.metrics.AddProcessedMessage(xatuEventType, networkStr)
		}

		return result, nil
	}

	var filteredMessages []FilteredMessageWithIndex

	// Check each message ID against the sharding configuration
	for i, messageID := range messageIDs {
		msgID := messageID.GetValue()
		if msgID == "" {
			continue
		}

		// Calculate the shard for this message ID
		shard := GetShard(msgID, gossipConfig.TotalShards)

		// Record metrics for all messages to track distribution
		m.metrics.AddShardObservation(xatuEventType, shard, networkStr)

		// Check if this shard is in the active shards list
		isActive := IsShardActive(shard, gossipConfig.ActiveShards)

		if isActive {
			filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
				MessageID:     messageID,
				OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
			})

			m.metrics.AddShardProcessed(xatuEventType, shard, networkStr)
			m.metrics.AddProcessedMessage(xatuEventType, networkStr)
		} else {
			m.metrics.AddShardSkipped(xatuEventType, shard, networkStr)
			m.metrics.AddSkippedMessage(xatuEventType, networkStr)
		}
	}

	return filteredMessages, nil
}

// filterRPCMetaMessagesHierarchical filters RPC meta messages using hierarchical configuration
func (m *Mimicry) filterRPCMetaMessagesHierarchical(
	messages []RPCMetaMessageInfo,
	topicConfig *TopicConfig,
	xatuEventType string,
	networkStr string,
) ([]FilteredMessageWithIndex, error) {
	var filteredMessages []FilteredMessageWithIndex

	// Process each message individually based on its gossip topic
	for i, msg := range messages {
		msgID := msg.MessageID.GetValue()
		if msgID == "" {
			continue
		}

		// Extract gossip topic from the message
		var gossipTopic string
		if msg.Topic != nil {
			gossipTopic = msg.Topic.GetValue()
		}

		// Get the appropriate gossip topic configuration
		gossipConfig, found := m.getGossipTopicConfig(topicConfig, gossipTopic)
		if !found {
			// No configuration found, skip this message
			m.metrics.AddSkippedMessage(xatuEventType, networkStr)

			continue
		}

		// If all shards are configured to be active, include the message
		//nolint:gosec // controlled config, no overflow.
		if len(gossipConfig.ActiveShards) == int(gossipConfig.TotalShards) {
			filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
				MessageID:     msg.MessageID,
				OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
			})

			m.metrics.AddProcessedMessage(xatuEventType, networkStr)

			continue
		}

		// Calculate the shard for this message ID
		shard := GetShard(msgID, gossipConfig.TotalShards)

		// Record metrics for all messages to track distribution
		m.metrics.AddShardObservation(xatuEventType, shard, networkStr)

		// Check if this shard is in the active shards list
		isActive := IsShardActive(shard, gossipConfig.ActiveShards)

		if isActive {
			filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
				MessageID:     msg.MessageID,
				OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
			})

			m.metrics.AddShardProcessed(xatuEventType, shard, networkStr)
			m.metrics.AddProcessedMessage(xatuEventType, networkStr)
		} else {
			m.metrics.AddShardSkipped(xatuEventType, shard, networkStr)
			m.metrics.AddSkippedMessage(xatuEventType, networkStr)
		}
	}

	return filteredMessages, nil
}

// filterRPCMetaMessagesWithSimpleConfig filters RPC meta messages using simple configuration
func (m *Mimicry) filterRPCMetaMessagesWithSimpleConfig(
	messageIDs []*wrapperspb.StringValue,
	topicConfig *TopicConfig,
	xatuEventType string,
	networkStr string,
) ([]FilteredMessageWithIndex, error) {
	totalShards := *topicConfig.TotalShards

	// If all shards are configured to be active, skip filtering
	//nolint:gosec // controlled config, no overflow.
	if len(topicConfig.ActiveShards) == int(totalShards) {
		result := make([]FilteredMessageWithIndex, len(messageIDs))

		for i, messageID := range messageIDs {
			result[i] = FilteredMessageWithIndex{
				MessageID:     messageID,
				OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
			}

			m.metrics.AddProcessedMessage(xatuEventType, networkStr)
		}

		return result, nil
	}

	var filteredMessages []FilteredMessageWithIndex

	// Check each message ID against the sharding configuration
	for i, messageID := range messageIDs {
		msgID := messageID.GetValue()
		if msgID == "" {
			continue
		}

		// Calculate the shard for this message ID
		shard := GetShard(msgID, totalShards)

		// Record metrics for all messages to track distribution
		m.metrics.AddShardObservation(xatuEventType, shard, networkStr)

		// Check if this shard is in the active shards list
		isActive := IsShardActive(shard, topicConfig.ActiveShards)

		if isActive {
			filteredMessages = append(filteredMessages, FilteredMessageWithIndex{
				MessageID:     messageID,
				OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
			})

			m.metrics.AddShardProcessed(xatuEventType, shard, networkStr)
			m.metrics.AddProcessedMessage(xatuEventType, networkStr)
		} else {
			m.metrics.AddShardSkipped(xatuEventType, shard, networkStr)
			m.metrics.AddSkippedMessage(xatuEventType, networkStr)
		}
	}

	return filteredMessages, nil
}

// shouldSkipRPCFiltering checks if RPC filtering should be skipped and returns early exit result
func (m *Mimicry) shouldSkipRPCFiltering(
	xatuEventType string,
	networkStr string,
	messageCount int,
	getMessageID func(int) *wrapperspb.StringValue,
) (bool, []FilteredMessageWithIndex) {
	if !m.Config.Traces.Enabled || isUnshardableEvent(xatuEventType) {
		return true, m.buildAllMessagesResult(messageCount, getMessageID, xatuEventType, networkStr)
	}

	return false, nil
}

// buildAllMessagesResult creates a result with all messages (no filtering)
func (m *Mimicry) buildAllMessagesResult(
	messageCount int,
	getMessageID func(int) *wrapperspb.StringValue,
	xatuEventType string,
	networkStr string,
) []FilteredMessageWithIndex {
	result := make([]FilteredMessageWithIndex, messageCount)

	for i := 0; i < messageCount; i++ {
		result[i] = FilteredMessageWithIndex{
			MessageID:     getMessageID(i),
			OriginalIndex: uint32(i), //nolint:gosec // conversion fine.
		}

		m.metrics.AddProcessedMessage(xatuEventType, networkStr)
	}

	return result
}

// getGossipTopicConfig finds the appropriate gossip topic configuration for a given gossip topic.
// Returns the matched configuration and true if found, or nil and false if no match.
func (m *Mimicry) getGossipTopicConfig(topicConfig *TopicConfig, gossipTopic string) (*GossipTopicConfig, bool) {
	if topicConfig.Topics == nil {
		return nil, false
	}

	// If we have a gossip topic, try to match it against patterns
	if gossipTopic != "" && topicConfig.Topics.GossipTopics != nil {
		// Check each gossip topic pattern directly
		for gossipPattern, gossipConfig := range topicConfig.Topics.GossipTopics {
			// Use regexp.MatchString for pattern matching
			if matched, err := regexp.MatchString(gossipPattern, gossipTopic); err == nil && matched {
				// Need to make a copy since we're returning a pointer to map value
				configCopy := gossipConfig

				return &configCopy, true
			}
		}
	}

	// Use fallback configuration if no pattern matched
	if topicConfig.Topics.Fallback != nil {
		return topicConfig.Topics.Fallback, true
	}

	return nil, false
}
