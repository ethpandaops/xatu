package canonical

import (
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconProposerDutyRoute struct{}

func (CanonicalBeaconProposerDutyRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_proposer_duty")
}

func (r CanonicalBeaconProposerDutyRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY).
		To(r.Table()).
		If(func(event *xatu.DecoratedEvent) bool {
			stateID := event.GetMeta().GetClient().GetEthV1ProposerDuty().GetStateId()

			return strings.EqualFold(stateID, "finalized")
		}).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.CopyFieldIfMissing("proposer_validator_index", "validator_index")).
		Apply(flattener.CopyFieldIfMissing("proposer_pubkey", "pubkey")).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(CanonicalBeaconProposerDutyRoute{})
}
