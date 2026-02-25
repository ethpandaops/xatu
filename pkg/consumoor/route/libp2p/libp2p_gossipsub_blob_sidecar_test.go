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

func TestSnapshot_libp2p_gossipsub_blob_sidecar(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/beacon_block/ssz_snappy"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubBlobSidecarBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR,
			DateTime: testfixture.TS(),
			Id:       "gs-blob-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubBlobSidecar{
				Libp2PTraceGossipsubBlobSidecar: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBlobSidecarData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					Topic:          wrapperspb.String(testTopic),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBlobSidecar{
			Libp2PTraceGossipsubBlobSidecar: &gossipsub.BlobSidecar{
				Index:         wrapperspb.UInt64(0),
				ProposerIndex: wrapperspb.UInt64(42),
			},
		},
	}, 1, map[string]any{
		"peer_id_unique_key": expectedPeerIDKey,
	})
}
