package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_pending_consolidation(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconStatePendingConsolidationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION,
			DateTime: testfixture.TS(),
			Id:       "pc-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStatePendingConsolidation{
				EthV1BeaconStatePendingConsolidation: &xatu.ClientMeta_AdditionalEthV1BeaconStatePendingConsolidationData{
					Epoch:           testfixture.EpochAdditional(),
					StateId:         "head",
					PositionInQueue: wrapperspb.UInt64(7),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStatePendingConsolidation{
			EthV1BeaconStatePendingConsolidation: &xatu.PendingConsolidationData{
				SourceIndex: wrapperspb.UInt64(100),
				TargetIndex: wrapperspb.UInt64(200),
			},
		},
	}, 1, map[string]any{
		"epoch":             uint32(3),
		"state_id":          "head",
		"position_in_queue": uint32(7),
		"source_index":      uint32(100),
		"target_index":      uint32(200),
	})
}
