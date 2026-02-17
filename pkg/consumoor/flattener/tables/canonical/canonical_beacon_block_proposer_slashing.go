package canonical

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlockProposerSlashingRoute struct{}

func (CanonicalBeaconBlockProposerSlashingRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_block_proposer_slashing")
}

func (r CanonicalBeaconBlockProposerSlashingRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.CopyFieldIfMissing("slot", "block_slot_number")).
		Apply(flattener.CopyFieldIfMissing("epoch", "block_epoch_number")).
		Apply(flattener.CopyFieldIfMissing("slot_start_date_time", "block_slot_start_date_time")).
		Apply(flattener.CopyFieldIfMissing("epoch_start_date_time", "block_epoch_start_date_time")).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(CanonicalBeaconBlockProposerSlashingRoute{})
}
