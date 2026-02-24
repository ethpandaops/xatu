package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_block(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2,
			DateTime: testfixture.TS(),
			Id:       "block-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsBlockV2{
				EthV1EventsBlockV2: &xatu.ClientMeta_AdditionalEthV1EventsBlockV2Data{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsBlockV2{
			EthV1EventsBlockV2: &ethv1.EventBlockV2{
				Slot:  wrapperspb.UInt64(100),
				Block: "0xblockroot",
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"block":             "0xblockroot",
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
