package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconSyncCommitteeRoute struct{}

func (CanonicalBeaconSyncCommitteeRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableCanonicalBeaconSyncCommittee,
		xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE,
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
		Mutator(flattener.SyncCommitteeMutator).
		Build()
}
