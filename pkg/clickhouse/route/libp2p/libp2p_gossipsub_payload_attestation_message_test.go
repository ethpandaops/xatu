package libp2p

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_gossipsub_payload_attestation_message(t *testing.T) {
	if len(libp2pGossipsubPayloadAttestationMessageEventNames) == 0 {
		t.Skip("no event names registered for libp2p_gossipsub_payload_attestation_message")
	}

	const (
		colValidatorIndex    = "validator_index"
		colPayloadPresent    = "payload_present"
		colBlobDataAvailable = "blob_data_available"
		colPeerIDUniqueKey   = "peer_id_unique_key"

		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/70b5f6d6/payload_attestation_message/ssz_snappy"
	)

	var (
		blockRoot         = repeatHex("5d", 32)
		expectedPeerIDKey = route.SeaHashInt64(testPeerID + testNetwork)
	)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubPayloadAttestationMessageBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     libp2pGossipsubPayloadAttestationMessageEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubPayloadAttestationMessage{
				Libp2PTraceGossipsubPayloadAttestationMessage: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubPayloadAttestationMessageData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					Topic:          wrapperspb.String(testTopic),
					MessageSize:    wrapperspb.UInt32(144),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubPayloadAttestationMessage{
			Libp2PTraceGossipsubPayloadAttestationMessage: &gossipsub.PayloadAttestationMessage{
				Slot:              wrapperspb.UInt64(48752),
				ValidatorIndex:    wrapperspb.UInt64(2891),
				BeaconBlockRoot:   wrapperspb.String(blockRoot),
				PayloadPresent:    wrapperspb.Bool(true),
				BlobDataAvailable: wrapperspb.Bool(true),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colValidatorIndex:             uint32(2891),
		colBeaconBlockRoot:            blockRoot,
		colPayloadPresent:             true,
		colBlobDataAvailable:          true,
		colPeerIDUniqueKey:            expectedPeerIDKey,
	})
}
