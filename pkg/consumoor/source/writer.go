package source

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Writer writes flattened rows to ClickHouse.
type Writer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Write(table string, event *xatu.DecoratedEvent, meta *metadata.CommonMetadata)
	// FlushAll forces all table writers to drain their buffers and write
	// to ClickHouse synchronously. Returns the first error encountered.
	// On failure, unflushed events are preserved in the table writers for
	// retry on the next cycle.
	FlushAll(ctx context.Context) error
}

// WriteErrorClassifier classifies sink write errors for source-level retry
// and reject handling.
type WriteErrorClassifier interface {
	IsPermanent(err error) bool
	Table(err error) string
}
