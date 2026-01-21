// Package cldata provides shared types and interfaces for consensus layer data processing.
package cldata

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/beacon/pkg/beacon"
)

// BeaconClient provides access to beacon node functionality needed by derivers.
// It abstracts the differences between how Cannon and Horizon interact with beacon nodes.
type BeaconClient interface {
	// GetBeaconBlock retrieves a beacon block by its identifier (slot number as string).
	// Returns nil without error if the block doesn't exist (missed slot).
	GetBeaconBlock(ctx context.Context, identifier string) (*spec.VersionedSignedBeaconBlock, error)

	// LazyLoadBeaconBlock queues a block for background preloading.
	// This is used for look-ahead optimization.
	LazyLoadBeaconBlock(identifier string)

	// Synced checks if the beacon node is synced and ready.
	// Returns an error if the node is not synced.
	Synced(ctx context.Context) error

	// Node returns the underlying beacon node for spec access.
	// This is needed for accessing fork epochs and slots per epoch.
	Node() beacon.Node

	// FetchBeaconBlockBlobs retrieves blob sidecars for a given block identifier.
	// Returns empty slice without error if no blobs exist for the slot.
	// This is used for Deneb+ blocks that contain blob transactions.
	FetchBeaconBlockBlobs(ctx context.Context, identifier string) ([]*deneb.BlobSidecar, error)
}
