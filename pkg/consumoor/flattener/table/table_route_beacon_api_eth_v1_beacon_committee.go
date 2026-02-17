package table

import (
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1BeaconCommitteeRoute struct{}

func (BeaconApiEthV1BeaconCommitteeRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1BeaconCommittee,
		xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
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
			stateID := event.GetMeta().GetClient().GetEthV1BeaconCommittee().GetStateId()

			return !strings.EqualFold(stateID, "finalized")
		}).
		Build()
}
