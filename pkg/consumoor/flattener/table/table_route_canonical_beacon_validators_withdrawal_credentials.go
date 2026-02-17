package table

import "github.com/ethpandaops/xatu/pkg/consumoor/flattener"

type CanonicalBeaconValidatorsWithdrawalCredentialsRoute struct{}

func (CanonicalBeaconValidatorsWithdrawalCredentialsRoute) Build() flattener.Route {
	return flattener.NewValidatorsFanoutFlattener(
		flattener.TableCanonicalBeaconValidatorsWithdrawalCredentials,
		flattener.ValidatorsFanoutKindWithdrawalCredential,
	)
}
