package source

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/consumoor/clickhouse"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Writer writes flattened rows to ClickHouse.
type Writer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	// FlushTableEvents writes the given events directly to their respective
	// ClickHouse tables concurrently. The map keys are base table names
	// (without suffix). Returns a FlushResult containing per-table errors
	// and any invalid events that should be sent to the DLQ.
	FlushTableEvents(ctx context.Context, tableEvents map[string][]*xatu.DecoratedEvent) *clickhouse.FlushResult
	// Ping checks connectivity to the underlying datastore.
	Ping(ctx context.Context) error
}
