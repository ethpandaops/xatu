package table

import "github.com/ethpandaops/xatu/pkg/consumoor/flattener"

type CanonicalBeaconValidatorsPubkeysRoute struct{}

func (CanonicalBeaconValidatorsPubkeysRoute) Build() flattener.Route {
	return flattener.NewValidatorsFanoutFlattener(flattener.TableCanonicalBeaconValidatorsPubkeys, flattener.ValidatorsFanoutKindPubkeys)
}
