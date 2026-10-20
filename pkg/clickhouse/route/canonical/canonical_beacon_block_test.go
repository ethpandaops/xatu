package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: testfixture.TS(),
			Id:       "cbb-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{
				EthV2BeaconBlockV2: &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{
					FinalizedWhenRequested: true,
					Slot:                   testfixture.SlotEpochAdditional(),
					Epoch:                  testfixture.EpochAdditional(),
					Version:                "deneb",
					BlockRoot:              "0xblockroot",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: &ethv2.EventBlockV2{
				Message: &ethv2.EventBlockV2_DenebBlock{
					DenebBlock: &ethv2.BeaconBlockDeneb{
						Slot:          wrapperspb.UInt64(100),
						ProposerIndex: wrapperspb.UInt64(42),
						ParentRoot:    "0xparentroot",
						StateRoot:     "0xstateroot",
						Body: &ethv2.BeaconBlockBodyDeneb{
							Eth1Data: &ethv1.Eth1Data{
								BlockHash:   "0xeth1blockhash",
								DepositRoot: "0xeth1depositroot",
							},
						},
					},
				},
			},
		},
	}, 1, map[string]any{
		"block_version": "deneb",
		"block_root":    "0xblockroot",
		"slot":          uint32(100),
	})
}

func gloasBlockEvent(additional *xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data) *xatu.DecoratedEvent {
	additional.FinalizedWhenRequested = true
	additional.Slot = testfixture.SlotEpochAdditional()
	additional.Epoch = testfixture.EpochAdditional()
	additional.Version = "gloas"
	additional.BlockRoot = "0xblockroot"

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: testfixture.TS(),
			Id:       "cbb-gloas",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{EthV2BeaconBlockV2: additional},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: &ethv2.EventBlockV2{
				Message: &ethv2.EventBlockV2_GloasBlock{
					GloasBlock: &ethv2.BeaconBlockGloas{
						Slot:          wrapperspb.UInt64(100),
						ProposerIndex: wrapperspb.UInt64(42),
						ParentRoot:    "0xparentroot",
						StateRoot:     "0xstateroot",
						Body: &ethv2.BeaconBlockBodyGloas{
							Eth1Data: &ethv1.Eth1Data{
								BlockHash:   "0xeth1blockhash",
								DepositRoot: "0xeth1depositroot",
							},
							SignedExecutionPayloadBid: &ethv1.SignedExecutionPayloadBid{
								Message: &ethv1.ExecutionPayloadBid{
									BuilderIndex: wrapperspb.UInt64(3),
									Value:        wrapperspb.UInt64(1000),
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestSnapshot_canonical_beacon_block_gloas_revealed_payload(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockBatch(), gloasBlockEvent(&xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{
		PayloadPresent:                      wrapperspb.Bool(true),
		ExecutionPayloadBlockAccessListHash: wrapperspb.String("0xbalhash"),
		TransactionsCount:                   wrapperspb.UInt64(2),
		ExecutionPayloadEnvelope: &ethv1.SignedExecutionPayloadEnvelope{
			Message: &ethv1.ExecutionPayloadEnvelope{
				Payload: &ethv1.ExecutionPayloadGloas{
					BlockHash:     "0xpayloadhash",
					ParentHash:    "0xpayloadparent",
					StateRoot:     "0xpayloadstate",
					FeeRecipient:  "0xfeerecipient",
					BlockNumber:   wrapperspb.UInt64(4242),
					GasLimit:      wrapperspb.UInt64(60_000_000),
					GasUsed:       wrapperspb.UInt64(21_000),
					BaseFeePerGas: "7",
					BlobGasUsed:   wrapperspb.UInt64(131_072),
					ExcessBlobGas: wrapperspb.UInt64(0),
					SlotNumber:    wrapperspb.UInt64(100),
				},
			},
		},
	}), 1, map[string]any{
		"block_version":                            "gloas",
		"payload_present":                          true,
		"builder_index":                            uint64(3),
		"bid_value":                                uint64(1000),
		"execution_payload_block_hash":             "0xpayloadhash",
		"execution_payload_parent_hash":            "0xpayloadparent",
		"execution_payload_state_root":             "0xpayloadstate",
		"execution_payload_fee_recipient":          "0xfeerecipient",
		"execution_payload_block_number":           uint32(4242),
		"execution_payload_gas_limit":              uint64(60_000_000),
		"execution_payload_gas_used":               uint64(21_000),
		"execution_payload_base_fee_per_gas":       "7",
		"execution_payload_blob_gas_used":          uint64(131_072),
		"execution_payload_excess_blob_gas":        uint64(0),
		"execution_payload_slot_number":            uint64(100),
		"execution_payload_block_access_list_root": "0xbalhash",
		"execution_payload_transactions_count":     uint32(2),
	})
}

func TestSnapshot_canonical_beacon_block_gloas_withheld_payload(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockBatch(), gloasBlockEvent(&xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{
		PayloadPresent: wrapperspb.Bool(false),
	}), 1, map[string]any{
		"payload_present":                          false,
		"builder_index":                            uint64(3),
		"execution_payload_block_hash":             nil,
		"execution_payload_gas_used":               nil,
		"execution_payload_base_fee_per_gas":       nil,
		"execution_payload_slot_number":            nil,
		"execution_payload_block_access_list_root": nil,
		"execution_payload_transactions_count":     nil,
	})
}
