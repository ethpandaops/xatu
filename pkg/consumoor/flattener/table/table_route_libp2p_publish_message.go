package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pPublishMessageRoute struct{}

func (Libp2pPublishMessageRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pPublishMessage, xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE).
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
