package beacon

import (
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1ProposerDutyRoute struct{}

func (BeaconApiEthV1ProposerDutyRoute) Table() flattener.TableName {
	return flattener.TableName("beacon_api_eth_v1_proposer_duty")
}

func (r BeaconApiEthV1ProposerDutyRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY).
		To(r.Table()).
		If(func(event *xatu.DecoratedEvent) bool {
			stateID := event.GetMeta().GetClient().GetEthV1ProposerDuty().GetStateId()

			return strings.EqualFold(stateID, "head")
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
	catalog.MustRegister(BeaconApiEthV1ProposerDutyRoute{})
}
