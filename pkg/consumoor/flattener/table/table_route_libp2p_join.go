package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pJoinRoute struct{}

func (Libp2pJoinRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pJoin, xatu.Event_LIBP2P_TRACE_JOIN).
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
