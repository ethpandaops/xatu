package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_join(t *testing.T) {
	const testTopic = "/eth2/bba4da96/beacon_block/ssz_snappy"

	testfixture.AssertSnapshot(t, newlibp2pJoinBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_JOIN,
			DateTime: testfixture.TS(),
			Id:       "join-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTraceJoin{
			Libp2PTraceJoin: &libp2ppb.Join{
				Topic: wrapperspb.String(testTopic),
			},
		},
	}, 1, map[string]any{
		"topic_layer":             "eth2",
		"topic_fork_digest_value": "bba4da96",
		"topic_name":              "beacon_block",
		"topic_encoding":          "ssz_snappy",
	})
}
