package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pPeerRoute struct{}

func (Libp2pPeerRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_peer")
}

func (r Libp2pPeerRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_CONNECTED,
			xatu.Event_LIBP2P_TRACE_DISCONNECTED,
			xatu.Event_LIBP2P_TRACE_ADD_PEER,
			xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
			xatu.Event_LIBP2P_TRACE_RECV_RPC,
			xatu.Event_LIBP2P_TRACE_SEND_RPC,
			xatu.Event_LIBP2P_TRACE_DROP_RPC,
			xatu.Event_LIBP2P_TRACE_GRAFT,
			xatu.Event_LIBP2P_TRACE_PRUNE).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.NormalizeDateTimeValues).
		Apply(EnrichFields).
		Mutator(UpsertPeerRow).
		Build()
}

func init() {
	catalog.MustRegister(Libp2pPeerRoute{})
}
