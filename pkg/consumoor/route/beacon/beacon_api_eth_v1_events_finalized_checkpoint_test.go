package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_finalized_checkpoint(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsFinalizedCheckpointBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2,
			DateTime: testfixture.TS(),
			Id:       "finalized-checkpoint-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsFinalizedCheckpointV2{
				EthV1EventsFinalizedCheckpointV2: &xatu.ClientMeta_AdditionalEthV1EventsFinalizedCheckpointV2Data{
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsFinalizedCheckpointV2{
			EthV1EventsFinalizedCheckpointV2: &ethv1.EventFinalizedCheckpointV2{
				Block: "0xfcblock",
				State: "0xfcstate",
				Epoch: wrapperspb.UInt64(3),
			},
		},
	}, 1, map[string]any{
		"block":             "0xfcblock",
		"state":             "0xfcstate",
		"epoch":             uint32(3),
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
