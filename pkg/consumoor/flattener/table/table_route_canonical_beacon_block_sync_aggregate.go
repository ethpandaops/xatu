package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlockSyncAggregateRoute struct{}

func (CanonicalBeaconBlockSyncAggregateRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableCanonicalBeaconBlockSyncAggregate,
		xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE,
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
		Mutator(flattener.SyncAggregateMutator).
		Build()
}
