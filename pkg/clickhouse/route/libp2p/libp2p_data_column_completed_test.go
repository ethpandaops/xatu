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

//nolint:goconst // map keys and literals are deliberately reused across sibling event tests
func TestSnapshot_libp2p_data_column_completed(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/data_column_sidecar_12/ssz_snappy"
	)

	blockRoot := "0x" + strings.Repeat("cd", 32)

	testfixture.AssertSnapshot(t, newlibp2pDataColumnCompletedBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_DATA_COLUMN_COMPLETED,
			DateTime: testfixture.TS(),
			Id:       "dcc-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceDataColumnCompleted{
				Libp2PTraceDataColumnCompleted: &xatu.ClientMeta_AdditionalLibP2PTraceDataColumnCompletedData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					Topic:          wrapperspb.String(testTopic),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceDataColumnCompleted{
			Libp2PTraceDataColumnCompleted: &gossipsub.DataColumnCompleted{
				BlockRoot:           wrapperspb.String(blockRoot),
				ColumnIndex:         wrapperspb.UInt64(12),
				CompletedVia:        wrapperspb.String("PARTIAL_GOSSIP"),
				CellsTotal:          wrapperspb.UInt32(9),
				CellsFromGossip:     wrapperspb.UInt32(4),
				CellsFromEl:         wrapperspb.UInt32(5),
				ContributingPeers:   wrapperspb.UInt32(2),
				KzgCommitmentsCount: wrapperspb.UInt32(9),
			},
		},
	}, 1, map[string]any{
		"slot":                        uint32(100),
		"propagation_slot_start_diff": uint32(500),
		"peer_id_unique_key":          route.SeaHashInt64(testPeerID + testNetwork),
		colBeaconBlockRoot:            blockRoot,
		colColumnIndex:                uint64(12),
		"completed_via":               "PARTIAL_GOSSIP",
		"cells_total":                 uint32(9),
		"cells_from_gossip":           uint32(4),
		"cells_from_el":               uint32(5),
		"contributing_peers":          uint32(2),
		"kzg_commitments_count":       uint32(9),
		"topic_name":                  "data_column_sidecar_12",
	})
}

// An EL-built column has no completing peer, so peer_id_unique_key is zero.
func TestSnapshot_libp2p_data_column_completed_NoPeer(t *testing.T) {
	testfixture.AssertSnapshot(t, newlibp2pDataColumnCompletedBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_DATA_COLUMN_COMPLETED,
			DateTime: testfixture.TS(),
			Id:       "dcc-2",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceDataColumnCompleted{
				Libp2PTraceDataColumnCompleted: &xatu.ClientMeta_AdditionalLibP2PTraceDataColumnCompletedData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceDataColumnCompleted{
			Libp2PTraceDataColumnCompleted: &gossipsub.DataColumnCompleted{
				ColumnIndex:  wrapperspb.UInt64(5),
				CompletedVia: wrapperspb.String("EL"),
				CellsTotal:   wrapperspb.UInt32(6),
				CellsFromEl:  wrapperspb.UInt32(6),
			},
		},
	}, 1, map[string]any{
		"peer_id_unique_key": int64(0),
		"completed_via":      "EL",
		"cells_from_el":      uint32(6),
		"topic_name":         "",
	})
}
