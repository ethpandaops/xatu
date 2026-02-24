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
	// FlushAll forces all table writers to drain their buffers and write
	// to ClickHouse synchronously. Returns a joined error containing all
	// table failures. On failure, unflushed events are preserved in the
	// table writers for retry on the next cycle.
	FlushAll(ctx context.Context) error
	// FlushTables forces the specified table writers (by base table name)
	// to drain their buffers and write to ClickHouse synchronously.
	// An empty or nil slice is a no-op that returns nil.
	FlushTables(ctx context.Context, tables []string) error
	// Ping checks connectivity to the underlying datastore.
	Ping(ctx context.Context) error
}

// WriteErrorClassifier classifies sink write errors for source-level retry
// and reject handling.
type WriteErrorClassifier interface {
	IsPermanent(err error) bool
	Table(err error) string
}
