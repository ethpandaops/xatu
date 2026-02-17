package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlockVoluntaryExitRoute struct{}

func (CanonicalBeaconBlockVoluntaryExitRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableCanonicalBeaconBlockVoluntaryExit,
		xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
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
