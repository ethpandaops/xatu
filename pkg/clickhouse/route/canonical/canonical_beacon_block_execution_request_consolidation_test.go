package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_execution_request_consolidation(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionRequestConsolidationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_CONSOLIDATION,
			DateTime: testfixture.TS(),
			Id:       "cberc-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionRequestConsolidation{
				EthV2BeaconBlockExecutionRequestConsolidation: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionRequestConsolidationData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
					PositionInBlock: wrapperspb.UInt64(2),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestConsolidation{
			EthV2BeaconBlockExecutionRequestConsolidation: &ethv1.ElectraExecutionRequestConsolidation{
				SourceAddress: wrapperspb.String("0x000000000000000000000000000000000000dead"),
				SourcePubkey:  wrapperspb.String("0xaaaa"),
				TargetPubkey:  wrapperspb.String("0xbbbb"),
			},
		},
	}, 1, map[string]any{
		"source_address":    "0x000000000000000000000000000000000000dead",
		"source_pubkey":     "0xaaaa",
		"target_pubkey":     "0xbbbb",
		"position_in_block": uint32(2),
	})
}
