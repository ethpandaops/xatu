package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_synthetic_builder_pending_payment_settlement(t *testing.T) {
	if len(beaconSyntheticBuilderPendingPaymentSettlementEventNames) == 0 {
		t.Skip("no event names registered for beacon_synthetic_builder_pending_payment_settlement")
	}

	const (
		colFeeRecipient = "fee_recipient"
		colAmount       = "amount"
		colWeight       = "weight"
		colQuorum       = "quorum"
		colOutcome      = "outcome"
		colEpoch        = "epoch"

		feeRecipient = "0x8943545177806ed17b9f23f0a21ee5948ecaa776"
	)

	testfixture.AssertSnapshot(t, newbeaconSyntheticBuilderPendingPaymentSettlementBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconSyntheticBuilderPendingPaymentSettlementEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_BeaconSyntheticBuilderPendingPaymentSettlement{
				BeaconSyntheticBuilderPendingPaymentSettlement: &xatu.ClientMeta_AdditionalBeaconSyntheticBuilderPendingPaymentSettlementData{
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_BeaconSyntheticBuilderPendingPaymentSettlement{
			BeaconSyntheticBuilderPendingPaymentSettlement: &ethv1.BuilderPendingPaymentSettlement{
				Epoch:        wrapperspb.UInt64(1523),
				BuilderIndex: wrapperspb.UInt64(2),
				FeeRecipient: feeRecipient,
				Amount:       wrapperspb.UInt64(1250000),
				Weight:       wrapperspb.UInt64(3400),
				Quorum:       wrapperspb.UInt64(2731),
				Outcome:      ethv1.BuilderPendingPaymentOutcome_BUILDER_PENDING_PAYMENT_OUTCOME_SETTLED,
				ResolvedAt:   testfixture.TS(),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colBuilderIndex:               uint64(2),
		colFeeRecipient:               feeRecipient,
		colAmount:                     uint64(1250000),
		colWeight:                     uint64(3400),
		colQuorum:                     uint64(2731),
		colOutcome:                    "SETTLED",
		colEpoch:                      uint32(3),
	})
}
