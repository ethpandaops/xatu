package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_deposit(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockDepositBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
			DateTime: testfixture.TS(),
			Id:       "cbd-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockDeposit{
				EthV2BeaconBlockDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockDepositData{
					Block: &xatu.BlockIdentifier{
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Version: "phase0",
						Root:    "0x1111111111111111111111111111111111111111111111111111111111111111",
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: &ethv1.DepositV2{
				Proof: []string{"0x2222222222222222222222222222222222222222222222222222222222222222"},
				Data: &ethv1.DepositV2_Data{
					Pubkey:                "0x933ad9491b62059dd065b560d256d8957a8c402cc6e8d8ee7290ae11e8f7329267a8811c397529dac52ae1342ba58c95",
					WithdrawalCredentials: "0x0033333333333333333333333333333333333333333333333333333333333333",
					Amount:                wrapperspb.UInt64(32000000000),
					Signature:             "0xa1b2c3",
				},
			},
		},
	}, 1, map[string]any{
		"meta_network_name":   "mainnet",
		"deposit_data_amount": "32000000000",
	})
}
