package libp2p

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

//nolint:goconst // map keys and literals are deliberately reused across sibling event tests
func TestSnapshot_libp2p_gossipsub_partial_data_column(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/data_column_sidecar_12/ssz_snappy"
		testEventID = "gs-pdc-1"
	)

	blockRoot := "0x" + strings.Repeat("cd", 32)
	groupID := "0x00" + strings.Repeat("cd", 32)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubPartialDataColumnBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN,
			DateTime: testfixture.TS(),
			Id:       testEventID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubPartialDataColumn{
				Libp2PTraceGossipsubPartialDataColumn: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubPartialDataColumnData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					Topic:          wrapperspb.String(testTopic),
					MessageSize:    wrapperspb.UInt32(4196),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubPartialDataColumn{
			Libp2PTraceGossipsubPartialDataColumn: &gossipsub.PartialDataColumn{
				Direction:              wrapperspb.String("RECV"),
				CapturePoint:           wrapperspb.String("WIRE"),
				GroupId:                wrapperspb.String(groupID),
				GroupVersion:           wrapperspb.UInt32(0),
				BlockRoot:              wrapperspb.String(blockRoot),
				ColumnIndex:            wrapperspb.UInt64(12),
				HasMessage:             wrapperspb.Bool(true),
				CellsCount:             wrapperspb.UInt32(2),
				CellsBitmap:            wrapperspb.String("0x4501"),
				BitmapLength:           wrapperspb.UInt32(9),
				HasHeader:              wrapperspb.Bool(true),
				KzgCommitmentsCount:    wrapperspb.UInt32(9),
				ProposerIndex:          wrapperspb.UInt64(1234),
				HasMetadata:            wrapperspb.Bool(true),
				MetadataSize:           wrapperspb.UInt32(8),
				MetadataAvailableCount: wrapperspb.UInt32(3),
				MetadataRequestsCount:  wrapperspb.UInt32(9),
				MetadataAvailable:      wrapperspb.String("0x0b02"),
				MetadataRequests:       wrapperspb.String("0xff03"),
			},
		},
	}, 1, map[string]any{
		"slot":                        uint32(100),
		"wallclock_slot":              uint32(100),
		"propagation_slot_start_diff": uint32(500),
		"direction":                   "RECV",
		"capture_point":               "WIRE",
		"peer_id_unique_key":          route.SeaHashInt64(testPeerID + testNetwork),
		"topic_name":                  "data_column_sidecar_12",
		"group_id":                    groupID,
		"group_version":               uint8(0),
		colBeaconBlockRoot:            blockRoot,
		colColumnIndex:                uint64(12),
		"has_message":                 true,
		"message_size":                uint32(4196),
		"cells_count":                 uint32(2),
		"cells_bitmap":                "0x4501",
		"bitmap_length":               uint32(9),
		"has_header":                  true,
		"kzg_commitments_count":       uint32(9),
		colProposerIndex:              uint32(1234),
		"has_metadata":                true,
		"metadata_size":               uint32(8),
		"metadata_available_count":    uint32(3),
		"metadata_requests_count":     uint32(9),
		"metadata_available":          "0x0b02",
		"metadata_requests":           "0xff03",
		"unique_key":                  route.SeaHashInt64(testEventID),
	})
}

// A metadata-only part whose block slot the capture point could not resolve:
// slot columns fall back to zero while the wallclock columns stay populated.
//
//nolint:goconst // map keys and literals are deliberately reused across sibling event tests
func TestSnapshot_libp2p_gossipsub_partial_data_column_UnknownSlot(t *testing.T) {
	testfixture.AssertSnapshot(t, newlibp2pGossipsubPartialDataColumnBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN,
			DateTime: testfixture.TS(),
			Id:       "gs-pdc-2",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubPartialDataColumn{
				Libp2PTraceGossipsubPartialDataColumn: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubPartialDataColumnData{
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Topic:          wrapperspb.String("/eth2/bba4da96/data_column_sidecar_3/ssz_snappy"),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String("16Uiu2HAmPeer2"),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubPartialDataColumn{
			Libp2PTraceGossipsubPartialDataColumn: &gossipsub.PartialDataColumn{
				Direction:    wrapperspb.String("SEND"),
				CapturePoint: wrapperspb.String("WIRE"),
				ColumnIndex:  wrapperspb.UInt64(3),
				HasMessage:   wrapperspb.Bool(false),
				HasMetadata:  wrapperspb.Bool(true),
				MetadataSize: wrapperspb.UInt32(8),
			},
		},
	}, 1, map[string]any{
		"slot":                        uint32(0),
		"slot_start_date_time":        int64(0),
		"epoch":                       uint32(0),
		"epoch_start_date_time":       int64(0),
		"propagation_slot_start_diff": uint32(0),
		"wallclock_slot":              uint32(100),
		"direction":                   "SEND",
		"has_message":                 false,
		"message_size":                uint32(0),
		colColumnIndex:                uint64(3),
	})
}

func TestFlatten_libp2p_gossipsub_partial_data_column_RequiresPeer(t *testing.T) {
	err := newlibp2pGossipsubPartialDataColumnBatch().FlattenTo(&xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN,
			DateTime: testfixture.TS(),
			Id:       "gs-pdc-3",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubPartialDataColumn{
				Libp2PTraceGossipsubPartialDataColumn: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubPartialDataColumnData{
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubPartialDataColumn{
			Libp2PTraceGossipsubPartialDataColumn: &gossipsub.PartialDataColumn{
				ColumnIndex: wrapperspb.UInt64(3),
			},
		},
	})
	require.ErrorIs(t, err, route.ErrInvalidEvent)
}
