// Package cldata provides shared types and interfaces for consensus layer data processing.
// It enables code reuse between the Cannon (historical backfill) and Horizon (real-time)
// modules by defining common abstractions.
package cldata

import (
	"context"

	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// ContextProvider supplies the contextual information needed by derivers
// to create properly decorated events. It abstracts the differences between
// Cannon and Horizon execution contexts.
type ContextProvider interface {
	// CreateClientMeta creates the client metadata for events.
	// This includes network information, client version, and other identifying data.
	CreateClientMeta(ctx context.Context) (*xatu.ClientMeta, error)

	// NetworkName returns the human-readable name of the network being monitored.
	NetworkName() string

	// NetworkID returns the numeric identifier of the network.
	NetworkID() uint64

	// Wallclock returns the Ethereum beacon chain wallclock for time calculations.
	// It provides slot and epoch timing information based on genesis time and slot duration.
	Wallclock() *ethwallclock.EthereumBeaconChain

	// DepositChainID returns the execution layer chain ID.
	// This is needed for transaction signing and verification.
	DepositChainID() uint64
}
