package source

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Writer writes flattened rows to ClickHouse.
type Writer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	// FlushTableEvents writes the given events directly to their respective
	// ClickHouse tables concurrently. The map keys are base table names
	// (without suffix). Returns a joined error containing all table failures.
	FlushTableEvents(ctx context.Context, tableEvents map[string][]*xatu.DecoratedEvent) error
	// Ping checks connectivity to the underlying datastore.
	Ping(ctx context.Context) error
}
