package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_chain_reorg(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsChainReorgBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2,
			DateTime: testfixture.TS(),
			Id:       "chain-reorg-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsChainReorgV2{
				EthV1EventsChainReorgV2: &xatu.ClientMeta_AdditionalEthV1EventsChainReorgV2Data{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsChainReorgV2{
			EthV1EventsChainReorgV2: &ethv1.EventChainReorgV2{
				Slot:  wrapperspb.UInt64(100),
				Depth: wrapperspb.UInt64(3),
				Epoch: wrapperspb.UInt64(3),
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"depth":             uint16(3),
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
