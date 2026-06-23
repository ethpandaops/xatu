package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_execution_request_withdrawal(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionRequestWithdrawalBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_WITHDRAWAL,
			DateTime: testfixture.TS(),
			Id:       "cberw-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionRequestWithdrawal{
				EthV2BeaconBlockExecutionRequestWithdrawal: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionRequestWithdrawalData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
					PositionInBlock: wrapperspb.UInt64(2),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestWithdrawal{
			EthV2BeaconBlockExecutionRequestWithdrawal: &ethv1.ElectraExecutionRequestWithdrawal{
				SourceAddress:   wrapperspb.String("0x000000000000000000000000000000000000dead"),
				ValidatorPubkey: wrapperspb.String("0xabcdef"),
				Amount:          wrapperspb.UInt64(32000000000),
			},
		},
	}, 1, map[string]any{
		"source_address":    "0x000000000000000000000000000000000000dead",
		"validator_pubkey":  "0xabcdef",
		"amount":            uint64(32000000000),
		"position_in_block": uint32(2),
	})
}
