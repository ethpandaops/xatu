package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlockExecutionTransactionRoute struct{}

func (CanonicalBeaconBlockExecutionTransactionRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableCanonicalBeaconBlockExecutionTransaction,
		xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
	).
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
