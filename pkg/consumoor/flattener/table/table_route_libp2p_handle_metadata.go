package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pHandleMetadataRoute struct{}

func (Libp2pHandleMetadataRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pHandleMetadata, xatu.Event_LIBP2P_TRACE_HANDLE_METADATA).
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
