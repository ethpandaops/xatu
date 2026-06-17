package libp2p

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Shared column names and topic field values for the join/leave snapshot tests.
const (
	colLocalPeerIDUniqueKey = "local_peer_id_unique_key"
	colTopicLayer           = "topic_layer"
	colTopicForkDigestValue = "topic_fork_digest_value"
	colTopicName            = "topic_name"
	colTopicEncoding        = "topic_encoding"
	valTopicLayer           = "eth2"
	valTopicName            = "beacon_block"
)

func TestSnapshot_libp2p_join(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmLocalHost"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/beacon_block/ssz_snappy"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pJoinBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_JOIN,
			DateTime: testfixture.TS(),
			Id:       "join-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceJoin{
				Libp2PTraceJoin: &xatu.ClientMeta_AdditionalLibP2PTraceJoinData{
					LocalPeerId: testPeerID,
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceJoin{
			Libp2PTraceJoin: &libp2ppb.Join{
				Topic: wrapperspb.String(testTopic),
			},
		},
	}, 1, map[string]any{
		colLocalPeerIDUniqueKey: expectedPeerIDKey,
		colTopicLayer:           valTopicLayer,
		colTopicForkDigestValue: "bba4da96",
		colTopicName:            valTopicName,
		colTopicEncoding:        "ssz_snappy",
	})
}
