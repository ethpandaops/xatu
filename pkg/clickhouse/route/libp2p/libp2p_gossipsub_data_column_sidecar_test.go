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

const (
	colColumnIndex     = "column_index"
	colProposerIndex   = "proposer_index"
	colBeaconBlockRoot = "beacon_block_root"
)

func TestSnapshot_libp2p_gossipsub_data_column_sidecar(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/beacon_block/ssz_snappy"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubDataColumnSidecarBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR,
			DateTime: testfixture.TS(),
			Id:       "gs-dcs-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubDataColumnSidecar{
				Libp2PTraceGossipsubDataColumnSidecar: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubDataColumnSidecarData{
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
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubDataColumnSidecar{
			Libp2PTraceGossipsubDataColumnSidecar: &gossipsub.DataColumnSidecar{
				Index:               wrapperspb.UInt64(1),
				ProposerIndex:       wrapperspb.UInt64(42),
				KzgCommitmentsCount: wrapperspb.UInt32(6),
			},
		},
	}, 1, map[string]any{
		"peer_id_unique_key": expectedPeerIDKey,
	})
}

// TestSnapshot_libp2p_gossipsub_data_column_sidecar_Gloas verifies that a Gloas
// data column sidecar event — which has no proposer_index or kzg_commitments_count
// because the signed block header is dropped (EIP-7732) — is still accepted and
// flattened, with the header-derived columns defaulting to zero.
func TestSnapshot_libp2p_gossipsub_data_column_sidecar_Gloas(t *testing.T) {
	const (
		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/bba4da96/data_column_sidecar_7/ssz_snappy"
	)

	expectedPeerIDKey := route.SeaHashInt64(testPeerID + testNetwork)
	blockRoot := "0x" + strings.Repeat("ab", 32)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubDataColumnSidecarBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR,
			DateTime: testfixture.TS(),
			Id:       "gs-dcs-gloas-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubDataColumnSidecar{
				Libp2PTraceGossipsubDataColumnSidecar: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubDataColumnSidecarData{
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
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubDataColumnSidecar{
			// Gloas: index and block_root only — no proposer_index / kzg_commitments_count.
			Libp2PTraceGossipsubDataColumnSidecar: &gossipsub.DataColumnSidecar{
				Index:     wrapperspb.UInt64(7),
				BlockRoot: wrapperspb.String(blockRoot),
			},
		},
	}, 1, map[string]any{
		"peer_id_unique_key":    expectedPeerIDKey,
		colColumnIndex:          uint64(7),
		colProposerIndex:        uint32(0),
		"kzg_commitments_count": uint32(0),
		colBeaconBlockRoot:      blockRoot,
	})
}
