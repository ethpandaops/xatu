package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_rpc_meta_subscription(t *testing.T) {
	const (
		testPeerID    = "16Uiu2HAmPeer1"
		testTopic     = "/eth2/bba4da96/beacon_block/ssz_snappy"
		testRootEvent = "root-event-456"
	)

	testfixture.AssertSnapshot(t, newlibp2pRpcMetaSubscriptionBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION,
			DateTime: testfixture.TS(),
			Id:       "sub-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaSubscription{
			Libp2PTraceRpcMetaSubscription: &libp2ppb.SubMetaItem{
				RootEventId:  wrapperspb.String(testRootEvent),
				PeerId:       wrapperspb.String(testPeerID),
				TopicId:      wrapperspb.String(testTopic),
				Subscribe:    wrapperspb.Bool(true),
				ControlIndex: wrapperspb.UInt32(0),
			},
		},
	}, 1, map[string]any{
		"rpc_meta_unique_key": route.SeaHashInt64(testRootEvent),
		"topic_layer":         "eth2",
		"topic_name":          "beacon_block",
	})
}
