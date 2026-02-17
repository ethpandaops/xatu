package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pConnectedRoute struct{}

func (Libp2pConnectedRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pConnected, xatu.Event_LIBP2P_TRACE_CONNECTED).
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
