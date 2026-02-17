package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlockDepositRoute struct{}

func (CanonicalBeaconBlockDepositRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableCanonicalBeaconBlockDeposit, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT).
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
