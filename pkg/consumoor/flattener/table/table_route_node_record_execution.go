package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type NodeRecordExecutionRoute struct{}

func (NodeRecordExecutionRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableNodeRecordExecution, xatu.Event_NODE_RECORD_EXECUTION).
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
