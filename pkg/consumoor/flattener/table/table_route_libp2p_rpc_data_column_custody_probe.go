package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcDataColumnCustodyProbeRoute struct{}

func (Libp2pRpcDataColumnCustodyProbeRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pRpcDataColumnCustodyProbe, xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE).
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
