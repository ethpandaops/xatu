package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
		"local_peer_id_unique_key": expectedPeerIDKey,
		"topic_layer":              "eth2",
		"topic_fork_digest_value":  "bba4da96",
		"topic_name":               "beacon_block",
		"topic_encoding":           "ssz_snappy",
	})
}
