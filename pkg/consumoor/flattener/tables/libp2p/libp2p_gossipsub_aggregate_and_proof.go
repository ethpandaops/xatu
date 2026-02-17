package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pGossipsubAggregateAndProofRoute struct{}

func (Libp2pGossipsubAggregateAndProofRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_gossipsub_aggregate_and_proof")
}

func (r Libp2pGossipsubAggregateAndProofRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.NormalizeDateTimeValues).
		Apply(EnrichFields).
		Build()
}

func init() {
	catalog.MustRegister(Libp2pGossipsubAggregateAndProofRoute{})
}
