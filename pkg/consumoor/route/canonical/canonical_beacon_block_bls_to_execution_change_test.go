package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_bls_to_execution_change(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockBlsToExecutionChangeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
			DateTime: testfixture.TS(),
			Id:       "cbbls-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockBlsToExecutionChange{
				EthV2BeaconBlockBlsToExecutionChange: &xatu.ClientMeta_AdditionalEthV2BeaconBlockBLSToExecutionChangeData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: &ethv2.SignedBLSToExecutionChangeV2{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
