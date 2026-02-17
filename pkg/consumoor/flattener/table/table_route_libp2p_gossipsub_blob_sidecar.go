package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pGossipsubBlobSidecarRoute struct{}

func (Libp2pGossipsubBlobSidecarRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pGossipsubBlobSidecar, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR).
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
