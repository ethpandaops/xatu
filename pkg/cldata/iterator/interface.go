// Package iterator provides shared interfaces for position tracking iterators.
// These interfaces abstract the position management for both Cannon (epoch-based)
// and Horizon (slot-based) modules.
package iterator

import (
	"context"
	"errors"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var (
	// ErrLocationUpToDate is returned when there is no new position to process.
	ErrLocationUpToDate = errors.New("location up to date")
)

// Direction indicates the processing direction of the iterator.
type Direction string

const (
	// DirectionForward processes positions moving forward (toward head).
	DirectionForward Direction = "forward"
	// DirectionBackward processes positions moving backward (backfill).
	DirectionBackward Direction = "backward"
)

// Position represents a position in the beacon chain that can be processed.
type Position struct {
	// Slot is the slot number to process.
	Slot phase0.Slot
	// Epoch is the epoch number to process (derived from slot if not set directly).
	Epoch phase0.Epoch
	// LookAheads contains upcoming positions for pre-fetching optimization.
	LookAheads []phase0.Slot
	// Direction indicates whether this is forward or backward processing.
	Direction Direction
}

// Iterator defines the interface for tracking and managing processing positions.
// It handles communication with the coordinator to persist progress and provides
// the next position to process.
type Iterator interface {
	// Start initializes the iterator with the activation fork version.
	// It should be called before Next() or UpdateLocation().
	Start(ctx context.Context, activationFork spec.DataVersion) error

	// Next returns the next position to process.
	// It blocks until a position is available or returns ErrLocationUpToDate
	// when caught up to head.
	Next(ctx context.Context) (*Position, error)

	// UpdateLocation persists the current position after successful processing.
	// This should be called after events have been successfully derived and sent.
	UpdateLocation(ctx context.Context, position *Position) error
}
