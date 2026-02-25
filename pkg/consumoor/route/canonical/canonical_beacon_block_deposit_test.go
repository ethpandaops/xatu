package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
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
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: &ethv1.DepositV2{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
