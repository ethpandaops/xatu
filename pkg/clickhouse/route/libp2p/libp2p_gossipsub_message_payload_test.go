package libp2p

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

//nolint:goconst // map keys and literals are deliberately reused across sibling event tests
func TestSnapshot_libp2p_gossipsub_message_payload(t *testing.T) {
	const (
		testPeerID    = "16Uiu2HAmPeer1"
		testNetwork   = "mainnet"
		testTopic     = "/eth2/bba4da96/beacon_block/ssz_snappy"
		testMessageID = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	)

	testData := []byte{0xff, 0x06, 0x00, 0x00, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x59}

	testfixture.AssertSnapshot(t, newlibp2pGossipsubMessagePayloadBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_MESSAGE_PAYLOAD,
			DateTime: testfixture.TS(),
			Id:       "gs-mp-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubMessagePayload{
				Libp2PTraceGossipsubMessagePayload: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubMessagePayloadData{
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Topic:          wrapperspb.String(testTopic),
					MessageId:      wrapperspb.String(testMessageID),
					MessageSize:    wrapperspb.UInt32(uint32(len(testData))),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubMessagePayload{
			Libp2PTraceGossipsubMessagePayload: &gossipsub.MessagePayload{
				Data:    testData,
				Outcome: wrapperspb.String("deliver"),
			},
		},
	}, 1, map[string]any{
		"message_id":   testMessageID,
		"message_data": string(testData),
	})
}
