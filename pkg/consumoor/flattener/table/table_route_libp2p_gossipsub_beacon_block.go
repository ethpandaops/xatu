package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pGossipsubBeaconBlockRoute struct{}

func (Libp2pGossipsubBeaconBlockRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pGossipsubBeaconBlock, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK).
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
