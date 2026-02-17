package canonical

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
)

type CanonicalBeaconValidatorsRoute struct{}

func (CanonicalBeaconValidatorsRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_validators")
}

func (r CanonicalBeaconValidatorsRoute) Build() flattener.Route {
	return newValidatorsFanoutRoute(r.Table(), validatorsFanoutKindValidators)
}

func init() {
	catalog.MustRegister(CanonicalBeaconValidatorsRoute{})
}
