package beacon

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsVoluntaryExitRoute struct{}

func (BeaconApiEthV1EventsVoluntaryExitRoute) Table() flattener.TableName {
	return flattener.TableName("beacon_api_eth_v1_events_voluntary_exit")
}

func (r BeaconApiEthV1EventsVoluntaryExitRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2).
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
	catalog.MustRegister(BeaconApiEthV1EventsVoluntaryExitRoute{})
}
