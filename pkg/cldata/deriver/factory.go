package deriver

import (
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/iterator"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

// DeriverFactory creates derivers from the registry.
type DeriverFactory struct {
	log         logrus.FieldLogger
	beacon      cldata.BeaconClient
	ctxProvider cldata.ContextProvider
}

// NewDeriverFactory creates a new deriver factory.
func NewDeriverFactory(
	log logrus.FieldLogger,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *DeriverFactory {
	return &DeriverFactory{
		log:         log,
		beacon:      beacon,
		ctxProvider: ctxProvider,
	}
}

// Create creates a generic deriver for the given cannon type.
// Returns nil if the cannon type is not registered.
func (f *DeriverFactory) Create(
	cannonType xatu.CannonType,
	enabled bool,
	iter iterator.Iterator,
) *GenericDeriver {
	spec, ok := Get(cannonType)
	if !ok {
		return nil
	}

	return NewGenericDeriver(
		f.log,
		spec,
		enabled,
		iter,
		f.beacon,
		f.ctxProvider,
	)
}

// CreateAll creates generic derivers for all registered types.
// The enabledFunc determines if each deriver should be enabled.
func (f *DeriverFactory) CreateAll(
	iterFactory func(cannonType xatu.CannonType) iterator.Iterator,
	enabledFunc func(cannonType xatu.CannonType) bool,
) []*GenericDeriver {
	specs := All()
	derivers := make([]*GenericDeriver, 0, len(specs))

	for _, spec := range specs {
		iter := iterFactory(spec.CannonType)
		if iter == nil {
			continue
		}

		enabled := enabledFunc(spec.CannonType)

		derivers = append(derivers, NewGenericDeriver(
			f.log,
			spec,
			enabled,
			iter,
			f.beacon,
			f.ctxProvider,
		))
	}

	return derivers
}

// RegisteredTypes returns all registered cannon types.
func RegisteredTypes() []xatu.CannonType {
	specs := All()
	types := make([]xatu.CannonType, 0, len(specs))

	for _, spec := range specs {
		types = append(types, spec.CannonType)
	}

	return types
}

// IsRegistered checks if a cannon type is registered.
func IsRegistered(cannonType xatu.CannonType) bool {
	_, ok := Get(cannonType)

	return ok
}
