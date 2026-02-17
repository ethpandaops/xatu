package execution

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionEngineNewPayloadRoute struct{}

func (ExecutionEngineNewPayloadRoute) Table() flattener.TableName {
	return flattener.TableName("execution_engine_new_payload")
}

func (r ExecutionEngineNewPayloadRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD).
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
	catalog.MustRegister(ExecutionEngineNewPayloadRoute{})
}
