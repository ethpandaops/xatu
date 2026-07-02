package libp2p

import (
	"math"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_gossipsub_execution_payload_envelope(t *testing.T) {
	if len(libp2pGossipsubExecutionPayloadEnvelopeEventNames) == 0 {
		t.Skip("no event names registered for libp2p_gossipsub_execution_payload_envelope")
	}

	const (
		colEnvelopeBlockRoot = "block_root"
		colBuilderIndex      = "builder_index"
		colBlockHash         = "block_hash"
		colPeerIDUniqueKey   = "peer_id_unique_key"

		testPeerID  = "16Uiu2HAmPeer1"
		testNetwork = "mainnet"
		testTopic   = "/eth2/70b5f6d6/execution_payload/ssz_snappy"

		// A self-built payload envelope: the proposer reveals its own payload,
		// so builder_index is the EIP-7732 self-build sentinel (max uint64).
		selfBuiltIndex = uint64(math.MaxUint64)
	)

	var (
		beaconBlockRoot   = repeatHex("6a", 32)
		blockHash         = repeatHex("7b", 32)
		stateRoot         = repeatHex("8c", 32)
		expectedPeerIDKey = route.SeaHashInt64(testPeerID + testNetwork)
	)

	testfixture.AssertSnapshot(t, newlibp2pGossipsubExecutionPayloadEnvelopeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     libp2pGossipsubExecutionPayloadEnvelopeEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_Libp2PTraceGossipsubExecutionPayloadEnvelope{
				Libp2PTraceGossipsubExecutionPayloadEnvelope: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubExecutionPayloadEnvelopeData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					Topic:          wrapperspb.String(testTopic),
					MessageSize:    wrapperspb.UInt32(1048576),
					Metadata: &libp2ppb.TraceEventMetadata{
						PeerId: wrapperspb.String(testPeerID),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubExecutionPayloadEnvelope{
			Libp2PTraceGossipsubExecutionPayloadEnvelope: &gossipsub.ExecutionPayloadEnvelope{
				Slot:            wrapperspb.UInt64(48752),
				BuilderIndex:    wrapperspb.UInt64(selfBuiltIndex),
				BeaconBlockRoot: wrapperspb.String(beaconBlockRoot),
				BlockHash:       wrapperspb.String(blockHash),
				StateRoot:       wrapperspb.String(stateRoot),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colEnvelopeBlockRoot:          beaconBlockRoot,
		colBuilderIndex:               selfBuiltIndex,
		colBlockHash:                  blockHash,
		colPeerIDUniqueKey:            expectedPeerIDKey,
	})
}
