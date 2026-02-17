package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pPruneRoute struct{}

func (Libp2pPruneRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pPrune, xatu.Event_LIBP2P_TRACE_PRUNE).
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
