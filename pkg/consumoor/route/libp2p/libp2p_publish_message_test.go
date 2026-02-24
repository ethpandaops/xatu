package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_publish_message(t *testing.T) {
	const testTopic = "/eth2/bba4da96/beacon_block/ssz_snappy"

	testfixture.AssertSnapshot(t, newlibp2pPublishMessageBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
			DateTime: testfixture.TS(),
			Id:       "pub-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_Libp2PTracePublishMessage{
			Libp2PTracePublishMessage: &libp2ppb.PublishMessage{
				Topic: wrapperspb.String(testTopic),
			},
		},
	}, 1, map[string]any{
		"topic_layer": "eth2",
		"topic_name":  "beacon_block",
	})
}
