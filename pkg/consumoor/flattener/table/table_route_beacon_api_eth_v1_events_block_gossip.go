package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsBlockGossipRoute struct{}

func (BeaconApiEthV1EventsBlockGossipRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableBeaconApiEthV1EventsBlockGossip, xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP).
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
