package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pSendRpcRoute struct{}

func (Libp2pSendRpcRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pSendRpc, xatu.Event_LIBP2P_TRACE_SEND_RPC).
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
