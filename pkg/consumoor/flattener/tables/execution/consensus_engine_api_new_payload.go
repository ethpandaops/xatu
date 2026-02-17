package execution

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ConsensusEngineApiNewPayloadRoute struct{}

func (ConsensusEngineApiNewPayloadRoute) Table() flattener.TableName {
	return flattener.TableName("consensus_engine_api_new_payload")
}

func (r ConsensusEngineApiNewPayloadRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD).
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
	catalog.MustRegister(ConsensusEngineApiNewPayloadRoute{})
}
