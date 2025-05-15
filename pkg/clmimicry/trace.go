package clmimicry

// ShouldTraceMessage determines whether a message with the given MsgID should be included
// in the sample based on the configured sampling settings.
func (m *Mimicry) ShouldTraceMessage(msgID, eventType string) bool {
	// If no msgID, we can't sample.
	if msgID == "" {
		return true
	}

	// Check if there's a matching topic config in the trace-based configuration.
	if m.Config.Traces.Enabled {
		topicConfig, found := m.Config.Traces.FindMatchingTopicConfig(eventType)
		if found {
			// Calculate the shard for this message.
			shard := GetShard(msgID, topicConfig.TotalShards)

			// Record metrics for all messages to track distribution.
			m.metrics.AddShardObservation(eventType, shard)

			// Check if this shard is in the active shards list.
			isActive := IsShardActive(shard, topicConfig.ActiveShards)

			// Record processed or skipped metrics.
			if isActive {
				m.metrics.AddShardProcessed(eventType, shard)
				m.metrics.AddProcessedMessage(eventType)
			} else {
				m.metrics.AddShardSkipped(eventType, shard)
				m.metrics.AddSkippedMessage(eventType)
			}

			return isActive
		}
	}

	// If no trace-based config matched, process all messages for enabled event types.
	m.metrics.AddProcessedMessage(eventType)

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
