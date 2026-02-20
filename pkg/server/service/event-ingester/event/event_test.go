package event_test

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event"
	"github.com/stretchr/testify/assert"
)

func TestEventRouter_AllTypesHaveHandlers(t *testing.T) {
	// Create a new EventRouter instance
	router := event.NewEventRouter(nil, nil, nil)

	// Event types that should be skipped (deprecated, unimplemented, or unknown types)
	skippedEventTypes := map[string]string{
		"BEACON_API_ETH_V1_EVENTS_UNKNOWN": "unknown event type",
		"LIBP2P_TRACE_UNKNOWN":             "unknown event type",
		"LIBP2P_TRACE_DROP_RPC":            "not implemented yet",
	}

	// List of all event types from event_ingester.proto
	eventTypes := xatu.Event_Name_name

	for _, eventType := range eventTypes {
		if reason, skip := skippedEventTypes[eventType]; skip {
			t.Logf("Skipping event type %s: %s", eventType, reason)
			continue
		}

		exists := router.HasRoute(event.Type(eventType))

		assert.True(t, exists, "Handler for event type %s does not exist", eventType)
	}
}
