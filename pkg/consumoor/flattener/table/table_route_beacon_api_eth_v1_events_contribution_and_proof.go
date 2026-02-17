package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsContributionAndProofRoute struct{}

func (BeaconApiEthV1EventsContributionAndProofRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1EventsContributionAndProof,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2,
	).
		CommonMetadata().
		RuntimeColumns().
		EventData().
		ClientAdditionalData().
		ServerAdditionalData().
		TableAliases().
		RouteAliases().
		NormalizeDateTimes().
		CommonEnrichment().
		Build()
}
