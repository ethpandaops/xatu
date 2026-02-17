package source

import (
	"context"
)

// Writer writes flattened rows to ClickHouse.
type Writer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Write(table string, rows []map[string]any)
	// FlushAll forces all table writers to drain their buffers and write
	// to ClickHouse synchronously. Returns the first error encountered.
	// On failure, unflushed rows are preserved in the table writers for
	// retry on the next cycle.
	FlushAll(ctx context.Context) error
}

// WriteErrorClassifier classifies sink write errors for source-level retry
// and reject handling.
type WriteErrorClassifier interface {
	IsPermanent(err error) bool
	Table(err error) string
}
