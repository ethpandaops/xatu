package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRemovePeerRoute struct{}

func (Libp2pRemovePeerRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRemovePeer, xatu.Event_LIBP2P_TRACE_REMOVE_PEER).
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
