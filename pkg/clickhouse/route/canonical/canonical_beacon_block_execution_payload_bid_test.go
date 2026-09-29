package canonical

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_execution_payload_bid(t *testing.T) {
	if len(canonicalBeaconBlockExecutionPayloadBidEventNames) == 0 {
		t.Skip("no event names registered for canonical_beacon_block_execution_payload_bid")
	}

	var (
		blockRoot       = "0x" + strings.Repeat("1a", 32)
		blockHash       = "0x" + strings.Repeat("2b", 32)
		parentBlockHash = "0x" + strings.Repeat("3c", 32)
		parentBlockRoot = "0x" + strings.Repeat("4d", 32)
		prevRandao      = "0x" + strings.Repeat("9f", 32)
		feeRecipient    = "0x8943545177806ed17b9f23f0a21ee5948ecaa776"
	)

	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionPayloadBidBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     canonicalBeaconBlockExecutionPayloadBidEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionPayloadBid{
				EthV2BeaconBlockExecutionPayloadBid: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionPayloadBidData{
					Block: &xatu.BlockIdentifier{
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Version: "gloas",
						Root:    blockRoot,
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionPayloadBid{
			EthV2BeaconBlockExecutionPayloadBid: &ethv1.SignedExecutionPayloadBid{
				Message: &ethv1.ExecutionPayloadBid{
					ParentBlockHash:  parentBlockHash,
					ParentBlockRoot:  parentBlockRoot,
					BlockHash:        blockHash,
					PrevRandao:       prevRandao,
					FeeRecipient:     feeRecipient,
					GasLimit:         wrapperspb.UInt64(60000000),
					BuilderIndex:     wrapperspb.UInt64(2),
					Slot:             wrapperspb.UInt64(48752),
					Value:            wrapperspb.UInt64(1250000),
					ExecutionPayment: wrapperspb.UInt64(1000000),
					BlobKzgCommitments: []string{
						"0x" + strings.Repeat("a0", 48),
						"0x" + strings.Repeat("a1", 48),
						"0x" + strings.Repeat("a2", 48),
					},
				},
				Signature: "0x" + strings.Repeat("5e", 96),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		"builder_index":               uint64(2),
		"block_hash":                  blockHash,
		"parent_block_hash":           parentBlockHash,
		"parent_block_root":           parentBlockRoot,
		"value":                       uint64(1250000),
		"execution_payment":           uint64(1000000),
		"fee_recipient":               feeRecipient,
		"gas_limit":                   uint64(60000000),
		"prev_randao":                 prevRandao,
		"blob_kzg_commitment_count":   uint32(3),
		"block_root":                  blockRoot,
		"block_version":               "gloas",
		"slot":                        uint32(100),
		"epoch":                       uint32(3),
	})
}
