package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_execution_request_deposit(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionRequestDepositBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_DEPOSIT,
			DateTime: testfixture.TS(),
			Id:       "cberd-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionRequestDeposit{
				EthV2BeaconBlockExecutionRequestDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionRequestDepositData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
					PositionInBlock: wrapperspb.UInt64(3),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestDeposit{
			EthV2BeaconBlockExecutionRequestDeposit: &ethv1.ElectraExecutionRequestDeposit{
				Pubkey:                wrapperspb.String("0xabc"),
				WithdrawalCredentials: wrapperspb.String("0xdef"),
				Amount:                wrapperspb.UInt64(32000000000),
				Signature:             wrapperspb.String("0x123"),
				Index:                 wrapperspb.UInt64(7),
			},
		},
	}, 1, map[string]any{
		"pubkey":                 "0xabc",
		"withdrawal_credentials": "0xdef",
		"amount":                 "32000000000",
		"signature":              "0x123",
		"index":                  uint64(7),
		"position_in_block":      uint32(3),
	})
}
