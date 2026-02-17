package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionEngineGetBlobsRoute struct{}

func (ExecutionEngineGetBlobsRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableExecutionEngineGetBlobs, xatu.Event_EXECUTION_ENGINE_GET_BLOBS).
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
