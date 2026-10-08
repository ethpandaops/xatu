package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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
		colSlot:             uint32(100),
		colBlock:            "0xblockroot",
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}

func TestSnapshot_beacon_api_eth_v1_events_block_gloas(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2,
			DateTime: testfixture.TS(),
			Id:       "block-gloas-1",
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
				Slot:                wrapperspb.UInt64(100),
				Block:               "0xblockroot",
				ExecutionOptimistic: true,
				BuilderIndex:        wrapperspb.UInt64(42),
				BlockHash:           "0xblockhash",
			},
		},
	}, 1, map[string]any{
		colSlot:                uint32(100),
		colBlock:               "0xblockroot",
		colExecutionOptimistic: true,
		"builder_index":        uint64(42),
		"block_hash":           "0xblockhash",
	})
}

// Pre-Gloas block events carry no builder_index or block_hash; both are stored
// as NULL rather than 0 / empty.
func TestSnapshot_beacon_api_eth_v1_events_block_pre_gloas_nulls(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2,
			DateTime: testfixture.TS(),
			Id:       "block-2",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
		Data: &xatu.DecoratedEvent_EthV1EventsBlockV2{
			EthV1EventsBlockV2: &ethv1.EventBlockV2{
				Slot:  wrapperspb.UInt64(100),
				Block: "0xblockroot",
			},
		},
	}, 1, map[string]any{
		"builder_index": nil,
		"block_hash":    nil,
	})
}
