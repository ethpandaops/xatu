package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pHandleStatusRoute struct{}

func (Libp2pHandleStatusRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pHandleStatus, xatu.Event_LIBP2P_TRACE_HANDLE_STATUS).
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
