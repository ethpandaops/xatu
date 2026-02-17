package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaControlIwantRoute struct{}

func (Libp2pRpcMetaControlIwantRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRpcMetaControlIwant, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT).
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
