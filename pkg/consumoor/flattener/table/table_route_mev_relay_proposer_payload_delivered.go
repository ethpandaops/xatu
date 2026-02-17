package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type MevRelayProposerPayloadDeliveredRoute struct{}

func (MevRelayProposerPayloadDeliveredRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableMevRelayProposerPayloadDelivered,
		xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED,
	).
		CommonMetadata().
		RuntimeColumns().
		EventData().
		ClientAdditionalData().
		ServerAdditionalData().
		TableAliases().
		RouteAliases().
		Aliases(map[string]string{"payload_delivered": ""}).
		NormalizeDateTimes().
		CommonEnrichment().
		Build()
}
