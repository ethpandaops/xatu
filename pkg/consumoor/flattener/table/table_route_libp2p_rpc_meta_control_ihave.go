package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaControlIhaveRoute struct{}

func (Libp2pRpcMetaControlIhaveRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRpcMetaControlIhave, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE).
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
