package node

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type NodeRecordExecutionRoute struct{}

func (NodeRecordExecutionRoute) Table() flattener.TableName {
	return flattener.TableName("node_record_execution")
}

func (r NodeRecordExecutionRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_NODE_RECORD_EXECUTION).
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
	catalog.MustRegister(NodeRecordExecutionRoute{})
}
