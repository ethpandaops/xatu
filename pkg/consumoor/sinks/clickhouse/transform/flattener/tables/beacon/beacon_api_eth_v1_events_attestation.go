package beacon

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsAttestationRoute struct{}

func (BeaconApiEthV1EventsAttestationRoute) Table() flattener.TableName {
	return flattener.TableName("beacon_api_eth_v1_events_attestation")
}

func (r BeaconApiEthV1EventsAttestationRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2).
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
	catalog.MustRegister(BeaconApiEthV1EventsAttestationRoute{})
}
