package execution

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ConsensusEngineApiGetBlobsRoute struct{}

func (ConsensusEngineApiGetBlobsRoute) Table() flattener.TableName {
	return flattener.TableName("consensus_engine_api_get_blobs")
}

func (r ConsensusEngineApiGetBlobsRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS).
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
	catalog.MustRegister(ConsensusEngineApiGetBlobsRoute{})
}
