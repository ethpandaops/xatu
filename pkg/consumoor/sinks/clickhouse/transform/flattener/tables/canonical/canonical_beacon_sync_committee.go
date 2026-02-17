package canonical

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconSyncCommitteeRoute struct{}

func (CanonicalBeaconSyncCommitteeRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_sync_committee")
}

func (r CanonicalBeaconSyncCommitteeRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.NormalizeDateTimeValues).
		Mutator(syncCommitteeMutator).
		Build()
}

func init() {
	catalog.MustRegister(CanonicalBeaconSyncCommitteeRoute{})
}
