package mev

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type MevRelayBidTraceRoute struct{}

func (MevRelayBidTraceRoute) Table() flattener.TableName {
	return flattener.TableName("mev_relay_bid_trace")
}

func (r MevRelayBidTraceRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.ApplyExplicitAliases(map[string]string{"bid_trace_builder_block_submission": ""})).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(MevRelayBidTraceRoute{})
}
