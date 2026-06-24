package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_pending_partial_withdrawal(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconStatePendingPartialWithdrawalBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL,
			DateTime: testfixture.TS(),
			Id:       "ppw-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStatePendingPartialWithdrawal{
				EthV1BeaconStatePendingPartialWithdrawal: &xatu.ClientMeta_AdditionalEthV1BeaconStatePendingPartialWithdrawalData{
					Epoch:           testfixture.EpochAdditional(),
					StateId:         "head",
					PositionInQueue: wrapperspb.UInt64(7),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStatePendingPartialWithdrawal{
			EthV1BeaconStatePendingPartialWithdrawal: &xatu.PendingPartialWithdrawalData{
				ValidatorIndex:    wrapperspb.UInt64(42),
				Amount:            wrapperspb.UInt64(1_000_000_000),
				WithdrawableEpoch: wrapperspb.UInt64(256),
			},
		},
	}, 1, map[string]any{
		"epoch":              uint32(3),
		"state_id":           "head",
		"position_in_queue":  uint32(7),
		"validator_index":    uint32(42),
		"amount":             "1000000000",
		"withdrawable_epoch": uint64(256),
		"meta_network_name":  "mainnet",
	})
}
