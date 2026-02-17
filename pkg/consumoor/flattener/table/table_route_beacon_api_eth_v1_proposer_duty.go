package table

import (
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1ProposerDutyRoute struct{}

func (BeaconApiEthV1ProposerDutyRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1ProposerDuty,
		xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
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
			stateID := event.GetMeta().GetClient().GetEthV1ProposerDuty().GetStateId()

			return strings.EqualFold(stateID, "head")
		}).
		Build()
}
