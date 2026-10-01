package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_builder_pending_payment(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconStateBuilderPendingPaymentBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_PAYMENT,
			DateTime: testfixture.TS(),
			Id:       "bpp-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStateBuilderPendingPayment{
				EthV1BeaconStateBuilderPendingPayment: &xatu.ClientMeta_AdditionalEthV1BeaconStateBuilderPendingPaymentData{
					Epoch:   testfixture.EpochAdditional(),
					Slot:    testfixture.SlotEpochAdditional(),
					StateId: "96",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStateBuilderPendingPayment{
			EthV1BeaconStateBuilderPendingPayment: &xatu.BuilderPendingPaymentData{
				PaymentIndex:  wrapperspb.UInt64(35),
				Weight:        wrapperspb.UInt64(64_000_000_000),
				FeeRecipient:  "0xf97e180c050e5ab072211ad2c213eb5aee4df134",
				Amount:        wrapperspb.UInt64(1_000_000),
				BuilderIndex:  wrapperspb.UInt64(7),
				ProposerIndex: wrapperspb.UInt64(83103),
			},
		},
	}, 1, map[string]any{
		"epoch":          uint32(3),
		"state_id":       "96",
		"payment_index":  uint32(35),
		"slot":           uint32(100),
		"weight":         uint64(64_000_000_000),
		"fee_recipient":  "0xf97e180c050e5ab072211ad2c213eb5aee4df134",
		"amount":         uint64(1_000_000),
		"builder_index":  uint64(7),
		"proposer_index": uint32(83103),
	})
}
