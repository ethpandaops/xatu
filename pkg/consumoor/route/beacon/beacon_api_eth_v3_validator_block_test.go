package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_api_eth_v3_validator_block(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV3ValidatorBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK,
			DateTime: testfixture.TS(),
			Id:       "v3-validator-block-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV3ValidatorBlock{
				EthV3ValidatorBlock: &xatu.ClientMeta_AdditionalEthV3ValidatorBlockData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV3ValidatorBlock{
			EthV3ValidatorBlock: &ethv2.EventBlockV2{},
		},
	}, 1, map[string]any{
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
