package clmimicry

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
)

func TestIsUnshardableEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expected  bool
	}{
		{
			name:      "JOIN event is unshardable",
			eventType: xatu.Event_LIBP2P_TRACE_JOIN.String(),
			expected:  true,
		},
		{
			name:      "ADD_PEER event is shardable",
			eventType: xatu.Event_LIBP2P_TRACE_ADD_PEER.String(),
			expected:  false,
		},
		{
			name:      "CONNECTED event is shardable",
			eventType: xatu.Event_LIBP2P_TRACE_CONNECTED.String(),
			expected:  false,
		},
		{
			name:      "Generic event is shardable",
			eventType: "SOME_OTHER_EVENT",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUnshardableEvent(tt.eventType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
