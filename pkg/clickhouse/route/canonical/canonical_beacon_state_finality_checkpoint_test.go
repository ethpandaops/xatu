package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_finality_checkpoint(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconStateFinalityCheckpointBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT,
			DateTime: testfixture.TS(),
			Id:       "fc-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStateFinalityCheckpoint{
				EthV1BeaconStateFinalityCheckpoint: &xatu.ClientMeta_AdditionalEthV1BeaconStateFinalityCheckpointData{
					Epoch:   testfixture.EpochAdditional(),
					StateId: "96",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStateFinalityCheckpoint{
			EthV1BeaconStateFinalityCheckpoint: &xatu.FinalityCheckpointData{
				PreviousJustified: &ethv1.Checkpoint{Epoch: 1, Root: "0xaaa"},
				CurrentJustified:  &ethv1.Checkpoint{Epoch: 2, Root: "0xbbb"},
				Finalized:         &ethv1.Checkpoint{Epoch: 0, Root: "0xccc"},
			},
		},
	}, 1, map[string]any{
		"epoch":                    uint32(3),
		"state_id":                 "96",
		"previous_justified_epoch": uint32(1),
		"previous_justified_root":  "0xaaa",
		"current_justified_epoch":  uint32(2),
		"current_justified_root":   "0xbbb",
		"finalized_epoch":          uint32(0),
		"finalized_root":           "0xccc",
	})
}
