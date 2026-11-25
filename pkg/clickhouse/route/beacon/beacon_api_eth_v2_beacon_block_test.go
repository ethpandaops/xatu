package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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
						Body: &ethv1.BeaconBlockBody{
							Graffiti: "0x4745613537394c48366462360000000000000000000000000000000000000000",
						},
					},
				},
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"graffiti_text":     "GEa579LH6db6",
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}

func TestSnapshot_beacon_api_eth_v2_beacon_block_pq(t *testing.T) {
	const (
		colProposerIndex = "proposer_index"
		colBlockVersion  = "block_version"
	)

	stateRoot := "0x38808cfb07e862c9f08120bf52ac8e0eaccfb654ae46bf638d9d006dd072e8b8"

	testfixture.AssertSnapshot(t, newbeaconApiEthV2BeaconBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: testfixture.TS(),
			Id:       "v2-beacon-block-pq",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{
				EthV2BeaconBlockV2: &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{
					Slot:    testfixture.SlotEpochAdditional(),
					Epoch:   testfixture.EpochAdditional(),
					Version: "pq",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: &ethv2.EventBlockV2{
				Version: ethv2.BlockVersion_PQ,
				Message: &ethv2.EventBlockV2_PqBlock{
					PqBlock: &ethv2.BeaconBlockPq{
						Slot:          wrapperspb.UInt64(10617),
						ProposerIndex: wrapperspb.UInt64(6961),
						StateRoot:     stateRoot,
						Body: &ethv2.BeaconBlockBodyPq{
							Graffiti: "0x6e696d6275732d676574682d3200000000000000000000000000000000000000",
							SignedExecutionPayloadBid: &ethv1.SignedExecutionPayloadBid{
								Message: &ethv1.ExecutionPayloadBid{
									BuilderIndex: wrapperspb.UInt64(5),
									Value:        wrapperspb.UInt64(1234),
								},
							},
						},
					},
				},
			},
		},
	}, 1, map[string]any{
		"slot":           uint32(10617),
		colProposerIndex: uint32(6961),
		"state_root":     stateRoot,
		colBlockVersion:  "pq",
		"graffiti_text":  "nimbus-geth-2",
		colBuilderIndex:  uint64(5),
		"bid_value":      uint64(1234),
	})
}
