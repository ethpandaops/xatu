package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pDuplicateMessageRoute struct{}

func (Libp2pDuplicateMessageRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pDuplicateMessage, xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE).
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
