package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_synthetic_heartbeat(t *testing.T) {
	testfixture.AssertSnapshot(t, newlibp2pSyntheticHeartbeatBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT,
			DateTime: testfixture.TS(),
			Id:       "hb-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceSyntheticHeartbeat{
			Libp2PTraceSyntheticHeartbeat: &libp2ppb.SyntheticHeartbeat{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
