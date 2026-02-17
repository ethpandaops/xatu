package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pPeerRoute struct{}

func (Libp2pPeerRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableLibp2pPeer,
		xatu.Event_LIBP2P_TRACE_CONNECTED,
		xatu.Event_LIBP2P_TRACE_DISCONNECTED,
		xatu.Event_LIBP2P_TRACE_ADD_PEER,
		xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
		xatu.Event_LIBP2P_TRACE_RECV_RPC,
		xatu.Event_LIBP2P_TRACE_SEND_RPC,
		xatu.Event_LIBP2P_TRACE_DROP_RPC,
		xatu.Event_LIBP2P_TRACE_GRAFT,
		xatu.Event_LIBP2P_TRACE_PRUNE,
	).
		CommonMetadata().
		RuntimeColumns().
		EventData().
		ClientAdditionalData().
		ServerAdditionalData().
		TableAliases().
		RouteAliases().
		NormalizeDateTimes().
		CommonEnrichment().
		Mutator(flattener.PeerConvergenceMutator).
		Build()
}
