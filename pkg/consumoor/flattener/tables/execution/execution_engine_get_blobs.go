package execution

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionEngineGetBlobsRoute struct{}

func (ExecutionEngineGetBlobsRoute) Table() flattener.TableName {
	return flattener.TableName("execution_engine_get_blobs")
}

func (r ExecutionEngineGetBlobsRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_EXECUTION_ENGINE_GET_BLOBS).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(ExecutionEngineGetBlobsRoute{})
}
