package clmimicry

import (
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/probe-lab/hermes/host"
)

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
			// Get the appropriate sharding key based on the configuration
			shardingKey := GetShardingKey(event, clientMeta, topicConfig.ShardingKey, xatuEventType)

			// If no sharding key, we can't sample. Shouldn't ever happen.
			if shardingKey == "" {
				m.metrics.AddProcessedMessage(xatuEventType, networkStr)

				return true
			}

			// If all shards are configured to be active, skip hashing and return true, save some trees.
			//nolint:gosec // controlled config, no overflow.
			if len(topicConfig.ActiveShards) == int(topicConfig.TotalShards) {
				m.metrics.AddProcessedMessage(xatuEventType, networkStr)

				return true
			}

			// Calculate the shard for this message.
			shard := GetShard(shardingKey, topicConfig.TotalShards)

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
	}

	// If no trace-based config matched, process all messages for enabled event types.
	m.metrics.AddProcessedMessage(xatuEventType, networkStr)

	return true
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
