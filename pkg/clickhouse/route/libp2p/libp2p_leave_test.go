package libp2p

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_leave(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmLocalHost"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/beacon_block/ssz_snappy"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pLeaveBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_LEAVE,
			DateTime: testfixture.TS(),
			Id:       "leave-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceLeave{
				Libp2PTraceLeave: &xatu.ClientMeta_AdditionalLibP2PTraceLeaveData{
					LocalPeerId: testPeerID,
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceLeave{
			Libp2PTraceLeave: &libp2ppb.Leave{
				Topic: wrapperspb.String(testTopic),
			},
		},
	}, 1, map[string]any{
		colLocalPeerIDUniqueKey: expectedPeerIDKey,
		colTopicLayer:           valTopicLayer,
		colTopicName:            valTopicName,
		colTopicEncoding:        "ssz_snappy",
	})
}
