package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionBlockMetricsRoute struct{}

func (ExecutionBlockMetricsRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableExecutionBlockMetrics, xatu.Event_EXECUTION_BLOCK_METRICS).
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
