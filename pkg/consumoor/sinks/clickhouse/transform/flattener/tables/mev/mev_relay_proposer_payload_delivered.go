package mev

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type MevRelayProposerPayloadDeliveredRoute struct{}

func (MevRelayProposerPayloadDeliveredRoute) Table() flattener.TableName {
	return flattener.TableName("mev_relay_proposer_payload_delivered")
}

func (r MevRelayProposerPayloadDeliveredRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.ApplyExplicitAliases(map[string]string{"payload_delivered": ""})).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(MevRelayProposerPayloadDeliveredRoute{})
}
