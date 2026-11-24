package libp2p

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

//nolint:goconst // map keys and literals are deliberately reused across sibling event tests
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
		"rpc_meta_unique_key":      route.SeaHashInt64(testRootEvent),
		"topic_layer":              "eth2",
		"topic_name":               "beacon_block",
		"requests_partial":         nil,
		"supports_sending_partial": nil,
	})
}

func TestSnapshot_libp2p_rpc_meta_subscription_PartialFlags(t *testing.T) {
	testfixture.AssertSnapshot(t, newlibp2pRpcMetaSubscriptionBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION,
			DateTime: testfixture.TS(),
			Id:       "sub-2",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaSubscription{
			Libp2PTraceRpcMetaSubscription: &libp2ppb.SubMetaItem{
				RootEventId:            wrapperspb.String("root-event-789"),
				PeerId:                 wrapperspb.String("16Uiu2HAmPeer1"),
				TopicId:                wrapperspb.String("/eth2/bba4da96/data_column_sidecar_4/ssz_snappy"),
				Subscribe:              wrapperspb.Bool(true),
				ControlIndex:           wrapperspb.UInt32(0),
				RequestsPartial:        wrapperspb.Bool(true),
				SupportsSendingPartial: wrapperspb.Bool(false),
			},
		},
	}, 1, map[string]any{
		"requests_partial":         true,
		"supports_sending_partial": false,
	})
}
