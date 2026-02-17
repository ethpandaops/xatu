package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pLeaveRoute struct{}

func (Libp2pLeaveRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pLeave, xatu.Event_LIBP2P_TRACE_LEAVE).
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
