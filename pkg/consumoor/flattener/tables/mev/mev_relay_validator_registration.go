package mev

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type MevRelayValidatorRegistrationRoute struct{}

func (MevRelayValidatorRegistrationRoute) Table() flattener.TableName {
	return flattener.TableName("mev_relay_validator_registration")
}

func (r MevRelayValidatorRegistrationRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(MevRelayValidatorRegistrationRoute{})
}
