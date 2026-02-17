package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaMessageRoute struct{}

func (Libp2pRpcMetaMessageRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRpcMetaMessage, xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE).
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
