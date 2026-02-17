package canonical

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
)

type CanonicalBeaconValidatorsPubkeysRoute struct{}

func (CanonicalBeaconValidatorsPubkeysRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_validators_pubkeys")
}

func (r CanonicalBeaconValidatorsPubkeysRoute) Build() flattener.Route {
	return newValidatorsFanoutRoute(r.Table(), validatorsFanoutKindPubkeys)
}

func init() {
	catalog.MustRegister(CanonicalBeaconValidatorsPubkeysRoute{})
}
