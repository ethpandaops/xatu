package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pDeliverMessageRoute struct{}

func (Libp2pDeliverMessageRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pDeliverMessage, xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE).
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
