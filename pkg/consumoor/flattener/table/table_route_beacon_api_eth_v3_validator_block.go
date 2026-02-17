package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV3ValidatorBlockRoute struct{}

func (BeaconApiEthV3ValidatorBlockRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableBeaconApiEthV3ValidatorBlock, xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK).
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
