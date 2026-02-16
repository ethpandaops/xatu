package consumoor

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisabledEventEnums(t *testing.T) {
	t.Run("parses valid names", func(t *testing.T) {
		cfg := &Config{
			DisabledEvents: []string{
				"BEACON_API_ETH_V1_EVENTS_HEAD",
				"LIBP2P_TRACE_CONNECTED",
			},
		}

		events, err := cfg.DisabledEventEnums()
		require.NoError(t, err)
		assert.Equal(
			t,
			[]xatu.Event_Name{
				xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				xatu.Event_LIBP2P_TRACE_CONNECTED,
			},
			events,
		)
	})

	t.Run("rejects unknown names", func(t *testing.T) {
		cfg := &Config{
			DisabledEvents: []string{
				"BEACON_API_ETH_V1_EVENTS_HEAD",
				"NOT_A_REAL_EVENT",
			},
		}

		_, err := cfg.DisabledEventEnums()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "NOT_A_REAL_EVENT")
	})
}
