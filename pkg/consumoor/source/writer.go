package source

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Writer writes flattened rows to ClickHouse.
type Writer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Write(table string, event *xatu.DecoratedEvent)
	// FlushTables forces the specified table writers (by base table name)
	// to drain their buffers and write to ClickHouse synchronously.
	// An empty or nil slice is a no-op that returns nil.
	FlushTables(ctx context.Context, tables []string) error
	// Ping checks connectivity to the underlying datastore.
	Ping(ctx context.Context) error
}
