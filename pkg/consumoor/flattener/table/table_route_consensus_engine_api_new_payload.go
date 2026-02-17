package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ConsensusEngineApiNewPayloadRoute struct{}

func (ConsensusEngineApiNewPayloadRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableConsensusEngineApiNewPayload, xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD).
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
