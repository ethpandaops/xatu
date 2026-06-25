package mev

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	mevrelay "github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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
				Slot:                 wrapperspb.UInt64(100),
				BlockNumber:          wrapperspb.UInt64(1000),
				BlockHash:            wrapperspb.String("0xblock"),
				BuilderPubkey:        wrapperspb.String("0xbuilder"),
				ProposerPubkey:       wrapperspb.String("0xproposer"),
				ProposerFeeRecipient: wrapperspb.String("0xfee"),
				GasLimit:             wrapperspb.UInt64(30000000),
				GasUsed:              wrapperspb.UInt64(15000000),
				Value:                wrapperspb.String("1000000000000000000"),
				NumTx:                wrapperspb.UInt64(50),
			},
		},
	}, 1, map[string]any{
		"slot":           uint32(100),
		"builder_pubkey": "0xbuilder",
	})
}
