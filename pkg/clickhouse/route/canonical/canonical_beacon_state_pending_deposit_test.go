package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_pending_deposit(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconStatePendingDepositBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT,
			DateTime: testfixture.TS(),
			Id:       "pd-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStatePendingDeposit{
				EthV1BeaconStatePendingDeposit: &xatu.ClientMeta_AdditionalEthV1BeaconStatePendingDepositData{
					Epoch:           testfixture.EpochAdditional(),
					StateId:         "head",
					PositionInQueue: wrapperspb.UInt64(7),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStatePendingDeposit{
			EthV1BeaconStatePendingDeposit: &xatu.PendingDepositData{
				Pubkey:                "0xpub",
				WithdrawalCredentials: "0xcreds",
				Amount:                wrapperspb.UInt64(32_000_000_000),
				Signature:             "0xsig",
				Slot:                  wrapperspb.UInt64(100),
			},
		},
	}, 1, map[string]any{
		"epoch":                  uint32(3),
		"state_id":               "head",
		"position_in_queue":      uint32(7),
		"pubkey":                 "0xpub",
		"withdrawal_credentials": "0xcreds",
		"amount":                 "32000000000",
		"signature":              "0xsig",
		"slot":                   uint32(100),
	})
}
