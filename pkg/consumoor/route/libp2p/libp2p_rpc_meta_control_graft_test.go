package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_rpc_meta_control_graft(t *testing.T) {
	const (
		testPeerID    = "16Uiu2HAmPeer1"
		testTopic     = "/eth2/bba4da96/beacon_block/ssz_snappy"
		testRootEvent = "root-event-456"
	)

	testfixture.AssertSnapshot(t, newlibp2pRpcMetaControlGraftBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT,
			DateTime: testfixture.TS(),
			Id:       "graft-meta-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlGraft{
			Libp2PTraceRpcMetaControlGraft: &libp2ppb.ControlGraftMetaItem{
				RootEventId:  wrapperspb.String(testRootEvent),
				PeerId:       wrapperspb.String(testPeerID),
				Topic:        wrapperspb.String(testTopic),
				ControlIndex: wrapperspb.UInt32(0),
			},
		},
	}, 1, map[string]any{
		"rpc_meta_unique_key": route.SeaHashInt64(testRootEvent),
		"topic_layer":         "eth2",
		"topic_name":          "beacon_block",
	})
}
