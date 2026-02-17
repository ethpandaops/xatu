package canonical

import (
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconCommitteeRoute struct{}

func (CanonicalBeaconCommitteeRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_committee")
}

func (r CanonicalBeaconCommitteeRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE).
		To(r.Table()).
		If(func(event *xatu.DecoratedEvent) bool {
			stateID := event.GetMeta().GetClient().GetEthV1BeaconCommittee().GetStateId()

			return strings.EqualFold(stateID, "finalized")
		}).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.CopyFieldIfMissing("committee_index", "index")).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(CanonicalBeaconCommitteeRoute{})
}
