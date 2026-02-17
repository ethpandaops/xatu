package node

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type NodeRecordConsensusRoute struct{}

func (NodeRecordConsensusRoute) Table() flattener.TableName {
	return flattener.TableName("node_record_consensus")
}

func (r NodeRecordConsensusRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_NODE_RECORD_CONSENSUS).
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
	catalog.MustRegister(NodeRecordConsensusRoute{})
}
