package beacon

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsBlobSidecarRoute struct{}

func (BeaconApiEthV1EventsBlobSidecarRoute) Table() flattener.TableName {
	return flattener.TableName("beacon_api_eth_v1_events_blob_sidecar")
}

func (r BeaconApiEthV1EventsBlobSidecarRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.CopyFieldIfMissing("column_index", "index")).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(BeaconApiEthV1EventsBlobSidecarRoute{})
}
