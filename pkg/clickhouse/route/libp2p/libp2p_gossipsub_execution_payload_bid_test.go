package libp2p

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// repeatHex builds a 0x-prefixed hex string from a repeated byte pair.
func repeatHex(pair string, count int) string {
	return "0x" + strings.Repeat(pair, count)
}

func TestSnapshot_libp2p_gossipsub_execution_payload_bid(t *testing.T) {
	if len(libp2pGossipsubExecutionPayloadBidEventNames) == 0 {
		t.Skip("no event names registered for libp2p_gossipsub_execution_payload_bid")
	}

	const (
		colBuilderIndex           = "builder_index"
		colBlockHash              = "block_hash"
		colValue                  = "value"
		colExecutionPayment       = "execution_payment"
		colFeeRecipient           = "fee_recipient"
		colGasLimit               = "gas_limit"
		colBlobKzgCommitmentCount = "blob_kzg_commitment_count"
		colPeerIDUniqueKey        = "peer_id_unique_key"

		testPeerID   = "16Uiu2HAmPeer1"
		testNetwork  = "mainnet"
		testTopic    = "/eth2/70b5f6d6/execution_payload_bid/ssz_snappy"
		feeRecipient = "0x8943545177806ed17b9f23f0a21ee5948ecaa776"
	)

	var (
		blockHash         = repeatHex("2b", 32)
		parentBlockHash   = repeatHex("3c", 32)
		expectedPeerIDKey = route.SeaHashInt64(testPeerID + testNetwork)
	)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubExecutionPayloadBidBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     libp2pGossipsubExecutionPayloadBidEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubExecutionPayloadBid{
				Libp2PTraceGossipsubExecutionPayloadBid: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubExecutionPayloadBidData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					Topic:          wrapperspb.String(testTopic),
					MessageSize:    wrapperspb.UInt32(320),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubExecutionPayloadBid{
			Libp2PTraceGossipsubExecutionPayloadBid: &gossipsub.ExecutionPayloadBid{
				Slot:                   wrapperspb.UInt64(48752),
				BuilderIndex:           wrapperspb.UInt64(2),
				BlockHash:              wrapperspb.String(blockHash),
				ParentBlockHash:        wrapperspb.String(parentBlockHash),
				Value:                  wrapperspb.UInt64(1250000),
				ExecutionPayment:       wrapperspb.UInt64(1000000),
				FeeRecipient:           wrapperspb.String(feeRecipient),
				GasLimit:               wrapperspb.UInt64(60000000),
				BlobKzgCommitmentCount: wrapperspb.UInt32(3),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colBuilderIndex:               uint64(2),
		colBlockHash:                  blockHash,
		"parent_block_hash":           parentBlockHash,
		colValue:                      uint64(1250000),
		colExecutionPayment:           uint64(1000000),
		colFeeRecipient:               feeRecipient,
		colGasLimit:                   uint64(60000000),
		colBlobKzgCommitmentCount:     uint32(3),
		colPeerIDUniqueKey:            expectedPeerIDKey,
	})
}
