package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlockRoute struct{}

func (CanonicalBeaconBlockRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableCanonicalBeaconBlock,
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
		Predicate(func(event *xatu.DecoratedEvent) bool {
			return event.GetMeta().GetClient().GetEthV2BeaconBlockV2().GetFinalizedWhenRequested()
		}).
		Build()
}
