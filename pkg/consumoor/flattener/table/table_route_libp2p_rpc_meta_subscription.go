package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaSubscriptionRoute struct{}

func (Libp2pRpcMetaSubscriptionRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRpcMetaSubscription, xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION).
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
