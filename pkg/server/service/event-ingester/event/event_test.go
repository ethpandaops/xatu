package event

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
)

func TestEventRouter_AllTypesHaveHandlers(t *testing.T) {
	// Create a new EventRouter instance
	router := NewEventRouter(nil, nil, nil)

	// List of all event types from event_ingester.proto
	eventTypes := xatu.Event_Name_name

	for _, eventType := range eventTypes {
		if eventType == "BEACON_API_ETH_V1_EVENTS_UNKNOWN" {
			continue
		}

		if eventType == "LIBP2P_TRACE_UNKNOWN" {
			continue
		}

		if eventType == "LIBP2P_TRACE_DROP_RPC" {
			// Not implemented yet
			continue
		}

		_, exists := router.routes[Type(eventType)]

		assert.True(t, exists, "Handler for event type %s does not exist", eventType)
	}
}
