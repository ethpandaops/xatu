package table

import "github.com/ethpandaops/xatu/pkg/consumoor/flattener"

type CanonicalBeaconValidatorsRoute struct{}

func (CanonicalBeaconValidatorsRoute) Build() flattener.Route {
	return flattener.NewValidatorsFanoutFlattener(flattener.TableCanonicalBeaconValidators, flattener.ValidatorsFanoutKindValidators)
}
