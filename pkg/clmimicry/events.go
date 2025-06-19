package clmimicry

import (
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// ShardingGroup represents the categorization of events based on their sharding capabilities
type ShardingGroup int

const (
	// GroupA events have both Topic and MsgID available for sharding
	GroupA ShardingGroup = iota
	// GroupB events have only Topic available for sharding
	GroupB
	// GroupC events have only MsgID available for sharding
	GroupC
	// GroupD events have no sharding keys available
	GroupD
)

// EventInfo contains metadata about an event type
type EventInfo struct {
	Type          xatu.Event_Name
	ShardingGroup ShardingGroup
	HasTopic      bool
	HasMsgID      bool
	IsMeta        bool // True for RPC meta events
}

// EventCategorizer manages event categorization for sharding decisions
type EventCategorizer struct {
	events map[xatu.Event_Name]*EventInfo
}

// NewEventCategorizer creates and initializes an EventCategorizer with all known events
func NewEventCategorizer() *EventCategorizer {
	ec := &EventCategorizer{
		events: make(map[xatu.Event_Name]*EventInfo),
	}
	ec.initializeEvents()

	return ec
}

// initializeEvents populates the event catalog with all known event types
func (ec *EventCategorizer) initializeEvents() {
	// Group A: Events with both Topic and MsgID
	ec.addEvent(xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE, GroupA, true, true, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE, GroupA, true, true, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE, GroupA, true, true, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE, GroupA, true, true, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK, GroupA, true, true, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION, GroupA, true, true, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR, GroupA, true, true, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE, GroupA, true, true, true)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE, GroupA, true, true, true)

	// Group B: Events with only Topic
	ec.addEvent(xatu.Event_LIBP2P_TRACE_JOIN, GroupB, true, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_LEAVE, GroupB, true, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_GRAFT, GroupB, true, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_PRUNE, GroupB, true, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT, GroupB, true, false, true)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE, GroupB, true, false, true)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION, GroupB, true, false, true)

	// Group C: Events with only MsgID
	ec.addEvent(xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT, GroupC, false, true, true)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT, GroupC, false, true, true)

	// Group D: Events with no sharding keys
	ec.addEvent(xatu.Event_LIBP2P_TRACE_ADD_PEER, GroupD, false, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_REMOVE_PEER, GroupD, false, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_RECV_RPC, GroupD, false, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_SEND_RPC, GroupD, false, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_DROP_RPC, GroupD, false, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_CONNECTED, GroupD, false, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_DISCONNECTED, GroupD, false, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_HANDLE_METADATA, GroupD, false, false, false)
	ec.addEvent(xatu.Event_LIBP2P_TRACE_HANDLE_STATUS, GroupD, false, false, false)
}

// addEvent is a helper to add event information
func (ec *EventCategorizer) addEvent(eventType xatu.Event_Name, group ShardingGroup, hasTopic, hasMsgID, isMeta bool) {
	ec.events[eventType] = &EventInfo{
		Type:          eventType,
		ShardingGroup: group,
		HasTopic:      hasTopic,
		HasMsgID:      hasMsgID,
		IsMeta:        isMeta,
	}
}

// GetEventInfo returns information about an event type
func (ec *EventCategorizer) GetEventInfo(eventType xatu.Event_Name) (*EventInfo, bool) {
	info, exists := ec.events[eventType]

	return info, exists
}

// GetShardingGroup returns the sharding group for an event type
func (ec *EventCategorizer) GetShardingGroup(eventType xatu.Event_Name) ShardingGroup {
	if info, exists := ec.events[eventType]; exists {
		return info.ShardingGroup
	}

	// Default to GroupD for unknown events
	return GroupD
}

// IsMetaEvent returns whether an event is an RPC meta event
func (ec *EventCategorizer) IsMetaEvent(eventType xatu.Event_Name) bool {
	if info, exists := ec.events[eventType]; exists {
		return info.IsMeta
	}

	return false
}

// GetAllEventsByGroup returns all events categorized by their sharding group
func (ec *EventCategorizer) GetAllEventsByGroup() map[ShardingGroup][]xatu.Event_Name {
	result := make(map[ShardingGroup][]xatu.Event_Name)

	for _, info := range ec.events {
		result[info.ShardingGroup] = append(result[info.ShardingGroup], info.Type)
	}

	return result
}

// GetGroupAEvents returns all events that have both Topic and MsgID
func (ec *EventCategorizer) GetGroupAEvents() []xatu.Event_Name {
	var events []xatu.Event_Name
	for _, info := range ec.events {
		if info.ShardingGroup == GroupA {
			events = append(events, info.Type)
		}
	}

	return events
}

// GetGroupBEvents returns all events that have only Topic
func (ec *EventCategorizer) GetGroupBEvents() []xatu.Event_Name {
	var events []xatu.Event_Name
	for _, info := range ec.events {
		if info.ShardingGroup == GroupB {
			events = append(events, info.Type)
		}
	}

	return events
}

// GetGroupCEvents returns all events that have only MsgID
func (ec *EventCategorizer) GetGroupCEvents() []xatu.Event_Name {
	var events []xatu.Event_Name
	for _, info := range ec.events {
		if info.ShardingGroup == GroupC {
			events = append(events, info.Type)
		}
	}

	return events
}

// GetGroupDEvents returns all events that have no sharding keys
func (ec *EventCategorizer) GetGroupDEvents() []xatu.Event_Name {
	var events []xatu.Event_Name
	for _, info := range ec.events {
		if info.ShardingGroup == GroupD {
			events = append(events, info.Type)
		}
	}

	return events
}
