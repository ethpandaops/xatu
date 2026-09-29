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

func TestSnapshot_libp2p_gossipsub_proposer_preferences(t *testing.T) {
	if len(libp2pGossipsubProposerPreferencesEventNames) == 0 {
		t.Skip("no event names registered for libp2p_gossipsub_proposer_preferences")
	}

	const (
		colValidatorIndex  = "validator_index"
		colFeeRecipient    = "fee_recipient"
		colTargetGasLimit  = "target_gas_limit"
		colPeerIDUniqueKey = "peer_id_unique_key"

		testPeerID   = "16Uiu2HAmPeer1"
		testNetwork  = "mainnet"
		testTopic    = "/eth2/70b5f6d6/proposer_preferences/ssz_snappy"
		feeRecipient = "0x8943545177806ed17b9f23f0a21ee5948ecaa776"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubProposerPreferencesBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     libp2pGossipsubProposerPreferencesEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubProposerPreferences{
				Libp2PTraceGossipsubProposerPreferences: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubProposerPreferencesData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					Topic:          wrapperspb.String(testTopic),
					MessageSize:    wrapperspb.UInt32(120),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubProposerPreferences{
			Libp2PTraceGossipsubProposerPreferences: &gossipsub.ProposerPreferences{
				Slot:           wrapperspb.UInt64(48752),
				ValidatorIndex: wrapperspb.UInt64(1337),
				FeeRecipient:   wrapperspb.String(feeRecipient),
				TargetGasLimit: wrapperspb.UInt64(60000000),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colValidatorIndex:             uint32(1337),
		colFeeRecipient:               feeRecipient,
		colTargetGasLimit:             uint64(60000000),
		colPeerIDUniqueKey:            expectedPeerIDKey,
	})
}
