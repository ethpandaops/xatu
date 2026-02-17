package canonical

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlockRoute struct{}

func (CanonicalBeaconBlockRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_block")
}

func (r CanonicalBeaconBlockRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2).
		To(r.Table()).
		If(func(event *xatu.DecoratedEvent) bool {
			return event.GetMeta().GetClient().GetEthV2BeaconBlockV2().GetFinalizedWhenRequested()
		}).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.CopyFieldsIfMissing(map[string]string{
			"slot":                                                  "slot_number",
			"epoch":                                                 "epoch_number",
			"eth1_data_block_hash":                                  "body_eth1_data_block_hash",
			"eth1_data_deposit_root":                                "body_eth1_data_deposit_root",
			"execution_payload_block_hash":                          "body_execution_payload_block_hash",
			"execution_payload_block_number":                        "body_execution_payload_block_number",
			"execution_payload_fee_recipient":                       "body_execution_payload_fee_recipient",
			"execution_payload_base_fee_per_gas":                    "body_execution_payload_base_fee_per_gas",
			"execution_payload_blob_gas_used":                       "body_execution_payload_blob_gas_used",
			"execution_payload_excess_blob_gas":                     "body_execution_payload_excess_blob_gas",
			"execution_payload_gas_limit":                           "body_execution_payload_gas_limit",
			"execution_payload_gas_used":                            "body_execution_payload_gas_used",
			"execution_payload_state_root":                          "body_execution_payload_state_root",
			"execution_payload_parent_hash":                         "body_execution_payload_parent_hash",
			"execution_payload_transactions_count":                  "transactions_count",
			"execution_payload_transactions_total_bytes":            "transactions_total_bytes",
			"execution_payload_transactions_total_bytes_compressed": "transactions_total_bytes_compressed",
		})).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(CanonicalBeaconBlockRoute{})
}
