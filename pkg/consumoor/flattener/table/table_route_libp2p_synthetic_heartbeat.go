package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pSyntheticHeartbeatRoute struct{}

func (Libp2pSyntheticHeartbeatRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pSyntheticHeartbeat, xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT).
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
