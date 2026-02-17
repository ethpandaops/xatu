package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsBlockRoute struct{}

func (BeaconApiEthV1EventsBlockRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1EventsBlock,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2,
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
