package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type MevRelayValidatorRegistrationRoute struct{}

func (MevRelayValidatorRegistrationRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableMevRelayValidatorRegistration, xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION).
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
