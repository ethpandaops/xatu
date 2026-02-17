package canonical

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlobSidecarRoute struct{}

func (CanonicalBeaconBlobSidecarRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_blob_sidecar")
}

func (r CanonicalBeaconBlobSidecarRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.CopyFieldIfMissing("blob_index", "index")).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(CanonicalBeaconBlobSidecarRoute{})
}
