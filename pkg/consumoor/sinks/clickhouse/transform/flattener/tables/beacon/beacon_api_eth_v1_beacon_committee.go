package beacon

import (
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1BeaconCommitteeRoute struct{}

func (BeaconApiEthV1BeaconCommitteeRoute) Table() flattener.TableName {
	return flattener.TableName("beacon_api_eth_v1_beacon_committee")
}

func (r BeaconApiEthV1BeaconCommitteeRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE).
		To(r.Table()).
		If(func(event *xatu.DecoratedEvent) bool {
			stateID := event.GetMeta().GetClient().GetEthV1BeaconCommittee().GetStateId()

			return !strings.EqualFold(stateID, "finalized")
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
	catalog.MustRegister(BeaconApiEthV1BeaconCommitteeRoute{})
}
