package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pGossipsubDataColumnSidecarRoute struct{}

func (Libp2pGossipsubDataColumnSidecarRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pGossipsubDataColumnSidecar, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR).
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
