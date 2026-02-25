package mev

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	mevrelay "github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_mev_relay_proposer_payload_delivered(t *testing.T) {
	testfixture.AssertSnapshot(t, newmevRelayProposerPayloadDeliveredBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED,
			DateTime: testfixture.TS(),
			Id:       "mevp-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_MevRelayPayloadDelivered{
				MevRelayPayloadDelivered: &xatu.ClientMeta_AdditionalMevRelayPayloadDeliveredData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_MevRelayPayloadDelivered{
			MevRelayPayloadDelivered: &mevrelay.ProposerPayloadDelivered{
				Slot:          wrapperspb.UInt64(100),
				BlockNumber:   wrapperspb.UInt64(1000),
				BuilderPubkey: wrapperspb.String("0xbuilder"),
				GasLimit:      wrapperspb.UInt64(30000000),
				GasUsed:       wrapperspb.UInt64(15000000),
				NumTx:         wrapperspb.UInt64(50),
			},
		},
	}, 1, map[string]any{
		"slot":           uint32(100),
		"builder_pubkey": "0xbuilder",
	})
}
