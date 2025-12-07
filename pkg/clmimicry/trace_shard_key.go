package clmimicry

// GetMsgID extracts the message ID from the event for sharding.
// We only shard based on message IDs, not peer IDs.
func GetMsgID(event TraceEvent) string {
	if event == nil {
		return ""
	}

	if me, ok := event.(MessageEvent); ok {
		return me.GetMsgID()
	}

	return ""
}

// GetGossipTopics extracts all gossip topics from a trace event if available.
// Returns a slice of unique topics found in the event.
func GetGossipTopics(event TraceEvent) []string {
	if event == nil {
		return nil
	}

	// Handle single-topic events first
	if te, ok := event.(TopicEvent); ok {
		topic := te.GetTopic()
		if topic != "" {
			return []string{topic}
		}
	}

	// Handle RPC events with multiple topics in RpcMeta
	if rpc, ok := event.(RPCMetaEvent); ok {
		return extractTopicsFromRPCMeta(rpc.GetRPCMeta())
	}

	return []string{}
}

// extractTopicsFromRPCMeta extracts unique topics from RpcMeta.
func extractTopicsFromRPCMeta(meta *RpcMeta) []string {
	if meta == nil {
		return []string{}
	}

	seen := make(map[string]struct{})
	topics := make([]string, 0)

	// Extract from Messages
	for _, msg := range meta.Messages {
		if msg.Topic != "" {
			if _, exists := seen[msg.Topic]; !exists {
				seen[msg.Topic] = struct{}{}
				topics = append(topics, msg.Topic)
			}
		}
	}

	// Extract from Subscriptions
	for _, sub := range meta.Subscriptions {
		if sub.TopicID != "" {
			if _, exists := seen[sub.TopicID]; !exists {
				seen[sub.TopicID] = struct{}{}
				topics = append(topics, sub.TopicID)
			}
		}
	}

	// Extract from Control messages
	if meta.Control != nil {
		// IHave
		for _, ihave := range meta.Control.IHave {
			if ihave.TopicID != "" {
				if _, exists := seen[ihave.TopicID]; !exists {
					seen[ihave.TopicID] = struct{}{}
					topics = append(topics, ihave.TopicID)
				}
			}
		}

		// IWant has no topic field

		// Graft
		for _, graft := range meta.Control.Graft {
			if graft.TopicID != "" {
				if _, exists := seen[graft.TopicID]; !exists {
					seen[graft.TopicID] = struct{}{}
					topics = append(topics, graft.TopicID)
				}
			}
		}

		// Prune
		for _, prune := range meta.Control.Prune {
			if prune.TopicID != "" {
				if _, exists := seen[prune.TopicID]; !exists {
					seen[prune.TopicID] = struct{}{}
					topics = append(topics, prune.TopicID)
				}
			}
		}

		// IDontWant has no topic field
	}

	return topics
}
