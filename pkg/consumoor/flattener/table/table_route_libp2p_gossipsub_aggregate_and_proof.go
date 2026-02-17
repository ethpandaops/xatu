package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pGossipsubAggregateAndProofRoute struct{}

func (Libp2pGossipsubAggregateAndProofRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pGossipsubAggregateAndProof, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF).
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
