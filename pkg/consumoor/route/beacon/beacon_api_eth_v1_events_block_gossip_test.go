package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_block_gossip(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsBlockGossipBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP,
			DateTime: testfixture.TS(),
			Id:       "block-gossip-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsBlockGossip{
				EthV1EventsBlockGossip: &xatu.ClientMeta_AdditionalEthV1EventsBlockGossipData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsBlockGossip{
			EthV1EventsBlockGossip: &ethv1.EventBlockGossip{
				Slot:  wrapperspb.UInt64(100),
				Block: "0xgossipblock",
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"block":             "0xgossipblock",
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
