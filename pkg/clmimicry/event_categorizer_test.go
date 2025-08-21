package clmimicry

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
)

func TestEventCategorization(t *testing.T) {
	ec := NewEventCategorizer()

	// Test all events are categorized
	allEventsByGroup := ec.GetAllEventsByGroup()

	totalEvents := 0
	for group, events := range allEventsByGroup {
		totalEvents += len(events)
		t.Logf("Group %d has %d events", group, len(events))
	}

	// We should have categorized all 28 known events
	assert.Equal(t, totalEvents, 28, "Should have exactly 27 events categorized")

	// Test specific group queries
	groupA := ec.GetGroupAEvents()
	assert.Len(t, groupA, 9, "Group A should have 9 events")

	groupB := ec.GetGroupBEvents()
	assert.Len(t, groupB, 7, "Group B should have 7 events")

	groupC := ec.GetGroupCEvents()
	assert.Len(t, groupC, 2, "Group C should have 2 events")

	groupD := ec.GetGroupDEvents()
	assert.GreaterOrEqual(t, len(groupD), 10, "Group D should have at least 10 events")
}

func TestMetaEventIdentification(t *testing.T) {
	ec := NewEventCategorizer()

	metaEvents := []xatu.Event_Name{
		xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE,
		xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION,
	}

	nonMetaEvents := []xatu.Event_Name{
		xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
		xatu.Event_LIBP2P_TRACE_JOIN,
		xatu.Event_LIBP2P_TRACE_ADD_PEER,
		xatu.Event_LIBP2P_TRACE_CONNECTED,
	}

	// Test meta events
	for _, event := range metaEvents {
		assert.True(t, ec.IsMetaEvent(event), "%s should be identified as meta event", event.String())
	}

	// Test non-meta events
	for _, event := range nonMetaEvents {
		assert.False(t, ec.IsMetaEvent(event), "%s should not be identified as meta event", event.String())
	}
}

func TestEventGroupCharacteristics(t *testing.T) {
	ec := NewEventCategorizer()

	tests := []struct {
		name          string
		events        []xatu.Event_Name
		expectTopic   bool
		expectMsgID   bool
		expectedGroup ShardingGroup
	}{
		{
			name: "Group A events have both Topic and MsgID",
			events: []xatu.Event_Name{
				xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
				xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE,
				xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			},
			expectTopic:   true,
			expectMsgID:   true,
			expectedGroup: GroupA,
		},
		{
			name: "Group B events have only Topic",
			events: []xatu.Event_Name{
				xatu.Event_LIBP2P_TRACE_JOIN,
				xatu.Event_LIBP2P_TRACE_LEAVE,
				xatu.Event_LIBP2P_TRACE_GRAFT,
			},
			expectTopic:   true,
			expectMsgID:   false,
			expectedGroup: GroupB,
		},
		{
			name: "Group C events have only MsgID",
			events: []xatu.Event_Name{
				xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
				xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT,
			},
			expectTopic:   false,
			expectMsgID:   true,
			expectedGroup: GroupC,
		},
		{
			name: "Group D events have neither",
			events: []xatu.Event_Name{
				xatu.Event_LIBP2P_TRACE_ADD_PEER,
				xatu.Event_LIBP2P_TRACE_CONNECTED,
				xatu.Event_LIBP2P_TRACE_RECV_RPC,
			},
			expectTopic:   false,
			expectMsgID:   false,
			expectedGroup: GroupD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, event := range tt.events {
				info, exists := ec.GetEventInfo(event)
				assert.True(t, exists, "Event %s should exist", event.String())
				assert.Equal(t, tt.expectTopic, info.HasTopic, "Event %s HasTopic mismatch", event.String())
				assert.Equal(t, tt.expectMsgID, info.HasMsgID, "Event %s HasMsgID mismatch", event.String())
				assert.Equal(t, tt.expectedGroup, info.ShardingGroup, "Event %s group mismatch", event.String())
			}
		})
	}
}

func TestEventCompleteness(t *testing.T) {
	ec := NewEventCategorizer()

	// List of all expected events
	allEvents := []xatu.Event_Name{
		// Group A
		xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
		xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE,
		xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE,
		xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE,
		xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
		xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION,
		xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR,
		xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE,
		// Group B
		xatu.Event_LIBP2P_TRACE_JOIN,
		xatu.Event_LIBP2P_TRACE_LEAVE,
		xatu.Event_LIBP2P_TRACE_GRAFT,
		xatu.Event_LIBP2P_TRACE_PRUNE,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE,
		xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION,
		// Group C
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
		xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT,
		// Group D
		xatu.Event_LIBP2P_TRACE_ADD_PEER,
		xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
		xatu.Event_LIBP2P_TRACE_RECV_RPC,
		xatu.Event_LIBP2P_TRACE_SEND_RPC,
		xatu.Event_LIBP2P_TRACE_DROP_RPC,
		xatu.Event_LIBP2P_TRACE_CONNECTED,
		xatu.Event_LIBP2P_TRACE_DISCONNECTED,
		xatu.Event_LIBP2P_TRACE_HANDLE_METADATA,
		xatu.Event_LIBP2P_TRACE_HANDLE_STATUS,
	}

	// Check all events are properly categorized
	for _, event := range allEvents {
		info, exists := ec.GetEventInfo(event)
		assert.True(t, exists, "Event %s should be categorized", event.String())
		assert.NotNil(t, info, "Event %s should have info", event.String())
		assert.Equal(t, event, info.Type, "Event type should match")
	}
}
