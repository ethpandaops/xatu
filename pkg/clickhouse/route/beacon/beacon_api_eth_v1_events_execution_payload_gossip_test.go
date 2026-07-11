package beacon

import (
	"math"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_api_eth_v1_events_execution_payload_gossip(t *testing.T) {
	if len(beaconApiEthV1EventsExecutionPayloadGossipEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_execution_payload_gossip")
	}

	// A self-built payload: the proposer reveals its own payload, so
	// builder_index is the EIP-7732 self-build sentinel (max uint64).
	const selfBuiltIndex = uint64(math.MaxUint64)

	var (
		beaconBlockRoot = repeatHex("6a", 32)
		blockHash       = repeatHex("7b", 32)
	)

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsExecutionPayloadGossipBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsExecutionPayloadGossipEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsExecutionPayloadGossip{
				EthV1EventsExecutionPayloadGossip: &xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadGossipData{
					Slot:        testfixture.SlotEpochAdditional(),
					Epoch:       testfixture.EpochAdditional(),
					Propagation: testfixture.PropagationAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayloadGossip{
			EthV1EventsExecutionPayloadGossip: &ethv1.ExecutionPayloadEvent{
				Slot:         wrapperspb.UInt64(48752),
				BuilderIndex: wrapperspb.UInt64(selfBuiltIndex),
				BlockHash:    blockHash,
				BlockRoot:    beaconBlockRoot,
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colBlockRoot:                  beaconBlockRoot,
		colBuilderIndex:               selfBuiltIndex,
		colExecBlockHash:              blockHash,
	})
}
