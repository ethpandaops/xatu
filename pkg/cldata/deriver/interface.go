// Package deriver provides shared interfaces for consensus layer data derivers.
// These interfaces are used by both Cannon (historical backfill) and Horizon (real-time) modules.
package deriver

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// EventDeriver defines the interface for deriving events from consensus layer data.
// Implementations process beacon chain data and emit decorated events.
type EventDeriver interface {
	// Start begins the deriver's processing loop.
	// It should block until the context is cancelled or an error occurs.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the deriver.
	Stop(ctx context.Context) error

	// Name returns a human-readable identifier for the deriver.
	Name() string

	// CannonType returns the CannonType that identifies the type of events this deriver produces.
	// Note: For Horizon derivers, this maps to the corresponding HorizonType.
	CannonType() xatu.CannonType

	// OnEventsDerived registers a callback to be invoked when events are derived.
	// Multiple callbacks can be registered and will be called in order.
	OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error)

	// ActivationFork returns the fork version at which this deriver becomes active.
	// Derivers should not process data from before their activation fork.
	ActivationFork() spec.DataVersion
}
