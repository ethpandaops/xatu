package execution

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionStateSizeRoute struct{}

func (ExecutionStateSizeRoute) Table() flattener.TableName {
	return flattener.TableName("execution_state_size")
}

func (r ExecutionStateSizeRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_EXECUTION_STATE_SIZE).
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
	catalog.MustRegister(ExecutionStateSizeRoute{})
}
