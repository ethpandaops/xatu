package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaControlIdontwantRoute struct{}

func (Libp2pRpcMetaControlIdontwantRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRpcMetaControlIdontwant, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT).
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
