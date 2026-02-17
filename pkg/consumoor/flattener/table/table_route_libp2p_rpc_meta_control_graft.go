package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaControlGraftRoute struct{}

func (Libp2pRpcMetaControlGraftRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRpcMetaControlGraft, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT).
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
