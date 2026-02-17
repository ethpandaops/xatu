package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionStateSizeRoute struct{}

func (ExecutionStateSizeRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableExecutionStateSize, xatu.Event_EXECUTION_STATE_SIZE).
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
