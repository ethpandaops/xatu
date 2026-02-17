package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pAddPeerRoute struct{}

func (Libp2pAddPeerRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pAddPeer, xatu.Event_LIBP2P_TRACE_ADD_PEER).
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
