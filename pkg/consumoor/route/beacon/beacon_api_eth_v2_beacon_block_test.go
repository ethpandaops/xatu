package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v2_beacon_block(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV2BeaconBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: testfixture.TS(),
			Id:       "v2-beacon-block-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{
				EthV2BeaconBlockV2: &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{
					FinalizedWhenRequested: false,
					Slot:                   testfixture.SlotEpochAdditional(),
					Epoch:                  testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: &ethv2.EventBlockV2{
				Message: &ethv2.EventBlockV2_Phase0Block{
					Phase0Block: &ethv1.BeaconBlockV2{
						Slot: wrapperspb.UInt64(100),
					},
				},
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
