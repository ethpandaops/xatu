package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// execution_optimistic used to be hard-coded false for these tables; it must
// now carry the beacon node's value.
func TestSnapshot_execution_optimistic_forwarded(t *testing.T) {
	t.Run("head", func(t *testing.T) {
		testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsHeadBatch(), &xatu.DecoratedEvent{
			Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2, DateTime: testfixture.TS(), Id: "head-optimistic"},
			Meta:  testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
			Data: &xatu.DecoratedEvent_EthV1EventsHeadV2{
				EthV1EventsHeadV2: &ethv1.EventHeadV2{Slot: wrapperspb.UInt64(100), ExecutionOptimistic: true},
			},
		}, 1, map[string]any{colExecutionOptimistic: true})
	})

	t.Run("finalized_checkpoint", func(t *testing.T) {
		testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsFinalizedCheckpointBatch(), &xatu.DecoratedEvent{
			Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2, DateTime: testfixture.TS(), Id: "fc-optimistic"},
			Meta:  testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
			Data: &xatu.DecoratedEvent_EthV1EventsFinalizedCheckpointV2{
				EthV1EventsFinalizedCheckpointV2: &ethv1.EventFinalizedCheckpointV2{Epoch: wrapperspb.UInt64(3), ExecutionOptimistic: true},
			},
		}, 1, map[string]any{colExecutionOptimistic: true})
	})

	t.Run("chain_reorg", func(t *testing.T) {
		testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsChainReorgBatch(), &xatu.DecoratedEvent{
			Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2, DateTime: testfixture.TS(), Id: "reorg-optimistic"},
			Meta:  testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
			Data: &xatu.DecoratedEvent_EthV1EventsChainReorgV2{
				EthV1EventsChainReorgV2: &ethv1.EventChainReorgV2{Slot: wrapperspb.UInt64(100), Depth: wrapperspb.UInt64(1), Epoch: wrapperspb.UInt64(3), ExecutionOptimistic: true},
			},
		}, 1, map[string]any{colExecutionOptimistic: true})
	})
}
