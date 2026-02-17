package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionEngineNewPayloadRoute struct{}

func (ExecutionEngineNewPayloadRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableExecutionEngineNewPayload, xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD).
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
