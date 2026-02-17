package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlockBlsToExecutionChangeRoute struct{}

func (CanonicalBeaconBlockBlsToExecutionChangeRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableCanonicalBeaconBlockBlsToExecutionChange,
		xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
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
