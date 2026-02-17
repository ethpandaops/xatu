package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsChainReorgRoute struct{}

func (BeaconApiEthV1EventsChainReorgRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1EventsChainReorg,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2,
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
