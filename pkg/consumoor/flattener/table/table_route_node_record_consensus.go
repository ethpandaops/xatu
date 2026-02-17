package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type NodeRecordConsensusRoute struct{}

func (NodeRecordConsensusRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableNodeRecordConsensus, xatu.Event_NODE_RECORD_CONSENSUS).
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
