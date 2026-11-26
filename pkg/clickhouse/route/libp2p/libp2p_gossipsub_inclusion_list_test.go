package libp2p

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_gossipsub_inclusion_list(t *testing.T) {
	if len(libp2pGossipsubInclusionListEventNames) == 0 {
		t.Skip("no event names registered for libp2p_gossipsub_inclusion_list")
	}

	const (
		colValidatorIndex         = "validator_index"
		colDependentRoot          = "dependent_root"
		colTransactionsCount      = "transactions_count"
		colTransactionsTotalBytes = "transactions_total_bytes"
		colTransactionHashes      = "transaction_hashes"
		colTopicName              = "topic_name"
		colPeerIDUniqueKey        = "peer_id_unique_key"

		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/70b5f6d6/inclusion_list/ssz_snappy"
	)

	var (
		dependentRoot     = repeatHex("7a", 32)
		txHashes          = []string{repeatHex("01", 32), repeatHex("02", 32)}
		expectedPeerIDKey = route.SeaHashInt64(testPeerID + testNetwork)
	)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubInclusionListBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     libp2pGossipsubInclusionListEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubInclusionList{
				Libp2PTraceGossipsubInclusionList: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubInclusionListData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					Topic:          wrapperspb.String(testTopic),
					MessageSize:    wrapperspb.UInt32(612),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubInclusionList{
			Libp2PTraceGossipsubInclusionList: &gossipsub.InclusionList{
				Slot:                   wrapperspb.UInt64(48752),
				ValidatorIndex:         wrapperspb.UInt64(731),
				DependentRoot:          wrapperspb.String(dependentRoot),
				TransactionsCount:      wrapperspb.UInt32(2),
				TransactionsTotalBytes: wrapperspb.UInt64(1450),
				TransactionHashes:      txHashes,
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colValidatorIndex:             uint32(731),
		colDependentRoot:              dependentRoot,
		colTransactionsCount:          uint32(2),
		colTransactionsTotalBytes:     uint32(1450),
		colTransactionHashes:          txHashes,
		colTopicName:                  "inclusion_list",
		colPeerIDUniqueKey:            expectedPeerIDKey,
	})
}
