package clmimicry

// ShouldTraceMessage determines whether a message with the given MsgID should be included
// in the sample based on the configured trace settings.
func (m *Mimicry) ShouldTraceMessage(msgID, eventType, network string) bool {
	// If no msgID, we can't sample.
	if msgID == "" {
		return true
	}

	// If network is empty, use unknown
	if network == "" {
		network = unknown
	}

	// Check if there's a matching topic config in the trace-based configuration.
	if m.Config.Traces.Enabled {
		topicConfig, found := m.Config.Traces.FindMatchingTopicConfig(eventType)
		if found {
			// If all shards are configured to be active, skip hashing and return true, some some trees.
			//nolint:gosec // controlled config, no overflow.
			if len(topicConfig.ActiveShards) == int(topicConfig.TotalShards) {
				m.metrics.AddProcessedMessage(eventType, network)

				return true
			}

			// Calculate the shard for this message.
			shard := GetShard(msgID, topicConfig.TotalShards)

			// Record metrics for all messages to track distribution.
			m.metrics.AddShardObservation(eventType, shard, network)

			// Check if this shard is in the active shards list.
			isActive := IsShardActive(shard, topicConfig.ActiveShards)

			// Record processed or skipped metrics.
			if isActive {
				m.metrics.AddShardProcessed(eventType, shard, network)
				m.metrics.AddProcessedMessage(eventType, network)
			} else {
				m.metrics.AddShardSkipped(eventType, shard, network)
				m.metrics.AddSkippedMessage(eventType, network)
			}

			return isActive
		}
	}

	// If no trace-based config matched, process all messages for enabled event types.
	m.metrics.AddProcessedMessage(eventType, network)

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
