package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_recv_rpc(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testEventID = "test-event-123"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pRecvRpcBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RECV_RPC,
			DateTime: testfixture.TS(),
			Id:       testEventID,
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceRecvRpc{
			Libp2PTraceRecvRpc: &libp2ppb.RecvRPC{
				PeerId: wrapperspb.String(testPeerID),
				Meta: &libp2ppb.RPCMeta{
					PeerId: wrapperspb.String(testPeerID),
				},
			},
		},
	}, 1, map[string]any{
		"unique_key":         route.SeaHashInt64(testEventID),
		"peer_id_unique_key": expectedPeerIDKey,
	})
}
