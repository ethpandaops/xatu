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
func TestSnapshot_libp2p_gossipsub_partial_data_column_outcome(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/data_column_sidecar_12/ssz_snappy"
		testEventID = "gs-pdco-1"
	)

	blockRoot := "0x" + strings.Repeat("cd", 32)
	groupID := "0x01" + strings.Repeat("cd", 32) + "6400000000000000"

	testfixture.AssertSnapshot(t, newlibp2pGossipsubPartialDataColumnOutcomeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME,
			DateTime: testfixture.TS(),
			Id:       testEventID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubPartialDataColumnOutcome{
				Libp2PTraceGossipsubPartialDataColumnOutcome: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubPartialDataColumnOutcomeData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Topic:          wrapperspb.String(testTopic),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubPartialDataColumnOutcome{
			Libp2PTraceGossipsubPartialDataColumnOutcome: &gossipsub.PartialDataColumnOutcome{
				GroupId:       wrapperspb.String(groupID),
				BlockRoot:     wrapperspb.String(blockRoot),
				ColumnIndex:   wrapperspb.UInt64(12),
				Stage:         wrapperspb.String("CELLS"),
				Outcome:       wrapperspb.String("REJECT"),
				Reason:        wrapperspb.String("invalid kzg proof"),
				CellsCount:    wrapperspb.UInt32(3),
				NewCellsCount: wrapperspb.UInt32(1),
			},
		},
	}, 1, map[string]any{
		"slot":               uint32(100),
		"epoch":              uint32(3),
		"wallclock_slot":     uint32(100),
		"peer_id_unique_key": route.SeaHashInt64(testPeerID + testNetwork),
		"topic_name":         "data_column_sidecar_12",
		"group_id":           groupID,
		colBeaconBlockRoot:   blockRoot,
		colColumnIndex:       uint64(12),
		"stage":              "CELLS",
		"outcome":            "REJECT",
		"reason":             "invalid kzg proof",
		"cells_count":        uint32(3),
		"new_cells_count":    uint32(1),
		"unique_key":         route.SeaHashInt64(testEventID),
	})
}
