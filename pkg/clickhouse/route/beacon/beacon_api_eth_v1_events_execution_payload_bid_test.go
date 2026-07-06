package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_api_eth_v1_events_execution_payload_bid(t *testing.T) {
	if len(beaconApiEthV1EventsExecutionPayloadBidEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_execution_payload_bid")
	}

	const (
		colParentBlockHash        = "parent_block_hash"
		colParentBlockRoot        = "parent_block_root"
		colValue                  = "value"
		colExecutionPayment       = "execution_payment"
		colFeeRecipient           = "fee_recipient"
		colGasLimit               = "gas_limit"
		colBlobKzgCommitmentCount = "blob_kzg_commitment_count"

		feeRecipient = "0x8943545177806ed17b9f23f0a21ee5948ecaa776"
	)

	var (
		blockHash       = repeatHex("2b", 32)
		parentBlockHash = repeatHex("3c", 32)
		parentBlockRoot = repeatHex("4d", 32)
	)

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsExecutionPayloadBidBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsExecutionPayloadBidEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsExecutionPayloadBid{
				EthV1EventsExecutionPayloadBid: &xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadBidData{
					Slot:        testfixture.SlotEpochAdditional(),
					Epoch:       testfixture.EpochAdditional(),
					Propagation: testfixture.PropagationAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayloadBid{
			EthV1EventsExecutionPayloadBid: &ethv1.SignedExecutionPayloadBid{
				Message: &ethv1.ExecutionPayloadBid{
					ParentBlockHash:  parentBlockHash,
					ParentBlockRoot:  parentBlockRoot,
					BlockHash:        blockHash,
					PrevRandao:       repeatHex("9f", 32),
					FeeRecipient:     feeRecipient,
					GasLimit:         wrapperspb.UInt64(60000000),
					BuilderIndex:     wrapperspb.UInt64(2),
					Slot:             wrapperspb.UInt64(48752),
					Value:            wrapperspb.UInt64(1250000),
					ExecutionPayment: wrapperspb.UInt64(1000000),
					BlobKzgCommitments: []string{
						repeatHex("a0", 48),
						repeatHex("a1", 48),
						repeatHex("a2", 48),
					},
				},
				Signature: repeatHex("51", 96),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colBuilderIndex:               uint64(2),
		colExecBlockHash:              blockHash,
		colParentBlockHash:            parentBlockHash,
		colParentBlockRoot:            parentBlockRoot,
		colValue:                      uint64(1250000),
		colExecutionPayment:           uint64(1000000),
		colFeeRecipient:               feeRecipient,
		colGasLimit:                   uint64(60000000),
		colBlobKzgCommitmentCount:     uint32(3),
	})
}

// Bids are broadcast before their slot begins, so the propagation delay is
// negative. It travels through the proto as a two's-complement uint64 and must
// land as a signed Int32, not a ~4.29e9 underflow.
func TestSnapshot_beacon_api_eth_v1_events_execution_payload_bid_preSlot(t *testing.T) {
	if len(beaconApiEthV1EventsExecutionPayloadBidEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_execution_payload_bid")
	}

	const colPropagationSlotStartDiff = "propagation_slot_start_diff"

	preSlotMs := int64(-400)

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsExecutionPayloadBidBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsExecutionPayloadBidEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsExecutionPayloadBid{
				EthV1EventsExecutionPayloadBid: &xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadBidData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
					Propagation: &xatu.PropagationV2{
						SlotStartDiff: wrapperspb.UInt64(uint64(preSlotMs)),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayloadBid{
			EthV1EventsExecutionPayloadBid: &ethv1.SignedExecutionPayloadBid{
				Message: &ethv1.ExecutionPayloadBid{
					BlockHash:    repeatHex("2b", 32),
					BuilderIndex: wrapperspb.UInt64(2),
					Slot:         wrapperspb.UInt64(48752),
				},
				Signature: repeatHex("51", 96),
			},
		},
	}, 1, map[string]any{
		colPropagationSlotStartDiff: int32(preSlotMs),
	})
}
