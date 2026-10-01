package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_builder_pending_withdrawal(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconStateBuilderPendingWithdrawalBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_WITHDRAWAL,
			DateTime: testfixture.TS(),
			Id:       "bpw-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStateBuilderPendingWithdrawal{
				EthV1BeaconStateBuilderPendingWithdrawal: &xatu.ClientMeta_AdditionalEthV1BeaconStateBuilderPendingWithdrawalData{
					Epoch:           testfixture.EpochAdditional(),
					StateId:         "96",
					PositionInQueue: wrapperspb.UInt64(4),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStateBuilderPendingWithdrawal{
			EthV1BeaconStateBuilderPendingWithdrawal: &xatu.BuilderPendingWithdrawalData{
				FeeRecipient: "0xf97e180c050e5ab072211ad2c213eb5aee4df134",
				Amount:       wrapperspb.UInt64(2_000_000),
				BuilderIndex: wrapperspb.UInt64(9),
			},
		},
	}, 1, map[string]any{
		"epoch":             uint32(3),
		"state_id":          "96",
		"position_in_queue": uint32(4),
		"fee_recipient":     "0xf97e180c050e5ab072211ad2c213eb5aee4df134",
		"amount":            uint64(2_000_000),
		"builder_index":     uint64(9),
	})
}
