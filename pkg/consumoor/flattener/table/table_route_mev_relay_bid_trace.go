package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type MevRelayBidTraceRoute struct{}

func (MevRelayBidTraceRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableMevRelayBidTrace,
		xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION,
	).
		CommonMetadata().
		RuntimeColumns().
		EventData().
		ClientAdditionalData().
		ServerAdditionalData().
		TableAliases().
		RouteAliases().
		Aliases(map[string]string{"bid_trace_builder_block_submission": ""}).
		NormalizeDateTimes().
		CommonEnrichment().
		Build()
}
