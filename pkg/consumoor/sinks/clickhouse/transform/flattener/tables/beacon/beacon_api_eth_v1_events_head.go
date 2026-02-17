package beacon

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsHeadRoute struct{}

func (BeaconApiEthV1EventsHeadRoute) Table() flattener.TableName {
	return flattener.TableName("beacon_api_eth_v1_events_head")
}

func (r BeaconApiEthV1EventsHeadRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2).
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
	catalog.MustRegister(BeaconApiEthV1EventsHeadRoute{})
}
