package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pDisconnectedRoute struct{}

func (Libp2pDisconnectedRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pDisconnected, xatu.Event_LIBP2P_TRACE_DISCONNECTED).
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
