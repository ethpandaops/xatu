package clmimicry

import (
	"reflect"

	"github.com/probe-lab/hermes/host"
)

// GetMsgID extracts the message ID from the event for sharding.
// We only shard based on message IDs, not peer IDs.
func GetMsgID(event *host.TraceEvent) string {
	if event == nil {
		return ""
	}

	// Handle map[string]any payloads (used by deliver_message, duplicate_message, etc.)
	if mapPayload, ok := event.Payload.(map[string]any); ok {
		if msgID, found := mapPayload["MsgID"]; found {
			if msgIDStr, ok := msgID.(string); ok {
				return msgIDStr
			}
		}

		return ""
	}

	// Try to access the MsgID field using reflection.
	v := reflect.ValueOf(event.Payload)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ""
	}

	// Dereference the pointer and check if it's a struct.
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return ""
	}

	// Try to find the MsgID field.
	msgIDField := v.FieldByName("MsgID")
	if !msgIDField.IsValid() || msgIDField.Kind() != reflect.String {
		return ""
	}

	return msgIDField.String()
}

// GetGossipTopics extracts all gossip topics from a trace event if available.
// Returns a slice of unique topics found in the event.
func GetGossipTopics(event *host.TraceEvent) []string {
	if event == nil {
		return nil
	}

	topicSet := make(map[string]bool)

	// First check if the event itself has a Topic field (used by gossipsub events)
	if event.Topic != "" {
		topicSet[event.Topic] = true
	}

	// Handle different payload types
	switch payload := event.Payload.(type) {
	case *host.RpcMeta:
		extractTopicsFromRpcMeta(payload, topicSet)
	case map[string]any:
		extractTopicsFromMapPayload(payload, topicSet)
	default:
		extractTopicsFromReflection(event.Payload, topicSet)
	}

	return convertTopicSetToSlice(topicSet)
}

// extractTopicsFromRpcMeta extracts topics from RPC meta payload.
func extractTopicsFromRpcMeta(rpcMeta *host.RpcMeta, topicSet map[string]bool) {
	// Extract topics from messages
	for _, msg := range rpcMeta.Messages {
		if msg.Topic != "" {
			topicSet[msg.Topic] = true
		}
	}

	// Extract topics from subscriptions
	for _, sub := range rpcMeta.Subscriptions {
		if sub.TopicID != "" {
			topicSet[sub.TopicID] = true
		}
	}

	// Extract topics from control messages
	if rpcMeta.Control != nil {
		extractTopicsFromControlMessages(rpcMeta.Control, topicSet)
	}
}

// extractTopicsFromControlMessages extracts topics from RPC control messages.
func extractTopicsFromControlMessages(control *host.RpcMetaControl, topicSet map[string]bool) {
	// Extract from IHave messages
	for _, ihave := range control.IHave {
		if ihave.TopicID != "" {
			topicSet[ihave.TopicID] = true
		}
	}

	// Extract from Graft messages
	for _, graft := range control.Graft {
		if graft.TopicID != "" {
			topicSet[graft.TopicID] = true
		}
	}

	// Extract from Prune messages
	for _, prune := range control.Prune {
		if prune.TopicID != "" {
			topicSet[prune.TopicID] = true
		}
	}
}

// extractTopicsFromMapPayload extracts topics from map-style payloads.
func extractTopicsFromMapPayload(mapPayload map[string]any, topicSet map[string]bool) {
	if topic, found := mapPayload["Topic"]; found {
		if topicStr, ok := topic.(string); ok && topicStr != "" {
			topicSet[topicStr] = true
		}
	}
}

// extractTopicsFromReflection uses reflection to extract topics from protobuf structures.
func extractTopicsFromReflection(payload interface{}, topicSet map[string]bool) {
	if payload == nil {
		return
	}

	v := reflect.ValueOf(payload)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}

	// Dereference the pointer and check if it's a struct
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}

	// Try to extract topic from both "Topic" and "TopicId" fields
	extractTopicFromField(v, "Topic", topicSet)
	extractTopicFromField(v, "TopicId", topicSet)
}

// extractTopicFromField extracts a topic from a specific struct field using reflection.
func extractTopicFromField(structValue reflect.Value, fieldName string, topicSet map[string]bool) {
	topicField := structValue.FieldByName(fieldName)
	if !topicField.IsValid() {
		return
	}

	// Check if the field type supports IsNil() before calling it
	switch topicField.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		if topicField.IsNil() {
			return
		}
	}

	// Handle *wrapperspb.StringValue fields
	if topicField.Kind() == reflect.Ptr && !topicField.IsNil() {
		// Check if it has a GetValue method (wrapperspb.StringValue)
		getValue := topicField.MethodByName("GetValue")
		if getValue.IsValid() {
			results := getValue.Call(nil)
			if len(results) > 0 && results[0].Kind() == reflect.String {
				topicStr := results[0].String()
				if topicStr != "" {
					topicSet[topicStr] = true
				}
			}
		}
	}
}

// convertTopicSetToSlice converts a topic set to a sorted slice.
func convertTopicSetToSlice(topicSet map[string]bool) []string {
	topics := make([]string, 0, len(topicSet))
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	return topics
}
