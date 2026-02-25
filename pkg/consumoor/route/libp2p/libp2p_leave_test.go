package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_leave(t *testing.T) {
	const testTopic = "/eth2/bba4da96/beacon_block/ssz_snappy"

	testfixture.AssertSnapshot(t, newlibp2pLeaveBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_LEAVE,
			DateTime: testfixture.TS(),
			Id:       "leave-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceLeave{
				Libp2PTraceLeave: &xatu.ClientMeta_AdditionalLibP2PTraceLeaveData{
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String("16Uiu2HAmPeer1"),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceLeave{
			Libp2PTraceLeave: &libp2ppb.Leave{
				Topic: wrapperspb.String(testTopic),
			},
		},
	}, 1, map[string]any{
		"topic_layer":    "eth2",
		"topic_name":     "beacon_block",
		"topic_encoding": "ssz_snappy",
	})
}
