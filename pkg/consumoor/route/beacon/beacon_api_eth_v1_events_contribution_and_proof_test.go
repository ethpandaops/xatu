package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_contribution_and_proof(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsContributionAndProofBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2,
			DateTime: testfixture.TS(),
			Id:       "contribution-and-proof-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsContributionAndProofV2{
				EthV1EventsContributionAndProofV2: &xatu.ClientMeta_AdditionalEthV1EventsContributionAndProofV2Data{},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsContributionAndProofV2{
			EthV1EventsContributionAndProofV2: &ethv1.EventContributionAndProofV2{
				Message: &ethv1.ContributionAndProofV2{
					AggregatorIndex: wrapperspb.UInt64(99),
				},
			},
		},
	}, 1, map[string]any{
		"aggregator_index":  uint32(99),
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
