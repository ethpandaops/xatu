package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_libp2p_gossipsub_beacon_block(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/beacon_block/ssz_snappy"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubBeaconBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			DateTime: testfixture.TS(),
			Id:       "gs-block-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubBeaconBlock{
				Libp2PTraceGossipsubBeaconBlock: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconBlockData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
					Topic: wrapperspb.String(testTopic),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconBlock{
			Libp2PTraceGossipsubBeaconBlock: &gossipsub.BeaconBlock{},
		},
	}, 1, map[string]any{
		"peer_id_unique_key": expectedPeerIDKey,
		"slot":               uint32(100),
		"epoch":              uint32(3),
		"topic_layer":        "eth2",
		"topic_name":         "beacon_block",
	})
}
