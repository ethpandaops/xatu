package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ConsensusEngineApiGetBlobsRoute struct{}

func (ConsensusEngineApiGetBlobsRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableConsensusEngineApiGetBlobs, xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS).
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
