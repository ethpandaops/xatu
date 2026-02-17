package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsVoluntaryExitRoute struct{}

func (BeaconApiEthV1EventsVoluntaryExitRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1EventsVoluntaryExit,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2,
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
