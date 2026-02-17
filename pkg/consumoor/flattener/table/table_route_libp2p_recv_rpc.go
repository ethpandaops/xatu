package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRecvRpcRoute struct{}

func (Libp2pRecvRpcRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRecvRpc, xatu.Event_LIBP2P_TRACE_RECV_RPC).
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
