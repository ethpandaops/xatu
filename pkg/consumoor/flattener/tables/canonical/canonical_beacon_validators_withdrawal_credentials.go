package canonical

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
)

type CanonicalBeaconValidatorsWithdrawalCredentialsRoute struct{}

func (CanonicalBeaconValidatorsWithdrawalCredentialsRoute) Table() flattener.TableName {
	return flattener.TableName("canonical_beacon_validators_withdrawal_credentials")
}

func (r CanonicalBeaconValidatorsWithdrawalCredentialsRoute) Build() flattener.Route {
	return newValidatorsFanoutRoute(r.Table(), validatorsFanoutKindWithdrawalCredential)
}

func init() {
	catalog.MustRegister(CanonicalBeaconValidatorsWithdrawalCredentialsRoute{})
}
