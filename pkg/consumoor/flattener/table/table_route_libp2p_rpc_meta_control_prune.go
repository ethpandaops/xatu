package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaControlPruneRoute struct{}

func (Libp2pRpcMetaControlPruneRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRpcMetaControlPrune, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE).
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
