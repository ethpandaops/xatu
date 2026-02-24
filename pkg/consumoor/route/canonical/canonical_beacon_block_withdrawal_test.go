package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_canonical_beacon_block_withdrawal(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockWithdrawalBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
			DateTime: testfixture.TS(),
			Id:       "cbw-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockWithdrawal{
				EthV2BeaconBlockWithdrawal: &xatu.ClientMeta_AdditionalEthV2BeaconBlockWithdrawalData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockWithdrawal{
			EthV2BeaconBlockWithdrawal: &ethv1.WithdrawalV2{
				Index:          wrapperspb.UInt64(10),
				ValidatorIndex: wrapperspb.UInt64(42),
			},
		},
	}, 1, map[string]any{
		"withdrawal_index":           uint32(10),
		"withdrawal_validator_index": uint32(42),
	})
}
