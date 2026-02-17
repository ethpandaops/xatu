package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pGraftRoute struct{}

func (Libp2pGraftRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pGraft, xatu.Event_LIBP2P_TRACE_GRAFT).
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
