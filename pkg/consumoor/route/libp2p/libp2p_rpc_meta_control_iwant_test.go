package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_rpc_meta_control_iwant(t *testing.T) {
	const (
		testPeerID    = "16Uiu2HAmPeer1"
		testNetwork   = "mainnet"
		testRootEvent = "root-event-456"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pRpcMetaControlIwantBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
			DateTime: testfixture.TS(),
			Id:       "iwant-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIwant{
			Libp2PTraceRpcMetaControlIwant: &libp2ppb.ControlIWantMetaItem{
				RootEventId:  wrapperspb.String(testRootEvent),
				PeerId:       wrapperspb.String(testPeerID),
				MessageIndex: wrapperspb.UInt32(0),
				ControlIndex: wrapperspb.UInt32(1),
			},
		},
	}, 1, map[string]any{
		"rpc_meta_unique_key": route.SeaHashInt64(testRootEvent),
		"peer_id_unique_key":  expectedPeerIDKey,
	})
}
