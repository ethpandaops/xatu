package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRejectMessageRoute struct{}

func (Libp2pRejectMessageRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRejectMessage, xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE).
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
