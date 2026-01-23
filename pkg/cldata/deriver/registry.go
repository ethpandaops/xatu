// Package deriver provides shared interfaces and a registry-based implementation
// for consensus layer data derivers. The registry pattern allows declarative
// definition of derivers, eliminating boilerplate code.
package deriver

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// ProcessingMode defines how a deriver processes data.
type ProcessingMode int

const (
	// ProcessingModeSlot processes data slot-by-slot within an epoch.
	// Used for block-based derivers that extract data from beacon blocks.
	ProcessingModeSlot ProcessingMode = iota

	// ProcessingModeEpoch processes data at the epoch level.
	// Used for derivers that fetch epoch-level data (committees, duties, etc.).
	ProcessingModeEpoch
)

// BlockExtractor extracts items from a beacon block for a slot-based deriver.
// Returns a slice of items that will each become a decorated event.
type BlockExtractor func(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error)

// EpochProcessor processes an entire epoch for epoch-based derivers.
// Returns all decorated events for the epoch.
type EpochProcessor func(
	ctx context.Context,
	epoch phase0.Epoch,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error)

// DeriverSpec defines the specification for a deriver type.
// This enables declarative registration of derivers without boilerplate.
type DeriverSpec struct {
	// Name is the human-readable name for logging.
	Name string

	// CannonType identifies the type of events this deriver produces.
	CannonType xatu.CannonType

	// ActivationFork is the fork at which this deriver becomes active.
	ActivationFork spec.DataVersion

	// Mode determines how this deriver processes data.
	Mode ProcessingMode

	// BlockExtractor extracts and creates events from a block (for slot mode).
	// Must be set if Mode is ProcessingModeSlot.
	BlockExtractor BlockExtractor

	// EpochProcessor processes an epoch (for epoch mode).
	// Must be set if Mode is ProcessingModeEpoch.
	EpochProcessor EpochProcessor
}

// registry holds all registered deriver specifications.
var registry = make(map[xatu.CannonType]*DeriverSpec)

// Register adds a deriver specification to the registry.
func Register(s *DeriverSpec) {
	registry[s.CannonType] = s
}

// Get retrieves a deriver specification by its cannon type.
func Get(cannonType xatu.CannonType) (*DeriverSpec, bool) {
	s, ok := registry[cannonType]

	return s, ok
}

// All returns all registered deriver specifications.
func All() []*DeriverSpec {
	specs := make([]*DeriverSpec, 0, len(registry))
	for _, spec := range registry {
		specs = append(specs, spec)
	}

	return specs
}
