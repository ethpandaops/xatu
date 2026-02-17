package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV2BeaconBlockRoute struct{}

func (BeaconApiEthV2BeaconBlockRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV2BeaconBlock,
		xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
		xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
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
