package kafkatopicrouter

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
)

func TestResolveTopic(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		eventName xatu.Event_Name
		want      string
	}{
		{
			name:      "kebab-case substitution",
			pattern:   "xatu-${event-name}",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			want:      "xatu-beacon-api-eth-v1-events-block",
		},
		{
			name:      "SCREAMING_SNAKE substitution",
			pattern:   "xatu-${EVENT_NAME}",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			want:      "xatu-BEACON_API_ETH_V1_EVENTS_BLOCK",
		},
		{
			name:      "both substitutions",
			pattern:   "${EVENT_NAME}-as-${event-name}",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
			want:      "BEACON_API_ETH_V1_EVENTS_HEAD-as-beacon-api-eth-v1-events-head",
		},
		{
			name:      "no variables (static topic)",
			pattern:   "static-topic",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			want:      "static-topic",
		},
		{
			name:      "prefix and suffix around kebab",
			pattern:   "prefix-${event-name}-suffix",
			eventName: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
			want:      "prefix-beacon-api-eth-v2-beacon-block-suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: tt.eventName,
				},
			}

			got := resolveTopic(tt.pattern, event)
			assert.Equal(t, tt.want, got)
		})
	}
}
