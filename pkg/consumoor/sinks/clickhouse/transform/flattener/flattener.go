package flattener

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// ColumnarBatch accumulates events directly into typed ch-go proto
// columns. Generated per-table Batch structs implement this interface.
type ColumnarBatch interface {
	// FlattenTo flattens one event into the batch's proto columns.
	FlattenTo(event *xatu.DecoratedEvent, meta *metadata.CommonMetadata) error
	// Input returns the proto.Input for the accumulated batch.
	Input() proto.Input
	// Reset clears all accumulated data.
	Reset()
	// Rows returns the number of rows appended so far.
	Rows() int
	// Snapshot reads back accumulated columnar data as row maps for
	// test assertions. Not used in production paths.
	Snapshot() []map[string]any
}

// Route converts a DecoratedEvent into flat ClickHouse rows for a
// specific target table. Each implementation handles one or more event
// names and produces rows for exactly one ClickHouse table.
type Route interface {
	// EventNames returns the event names this route handles.
	EventNames() []xatu.Event_Name

	// TableName returns the target ClickHouse table name.
	TableName() string

	// ShouldProcess returns whether this event should be processed by
	// this route. Used for conditional routing where the same event
	// name routes to different tables based on additional_data fields.
	ShouldProcess(event *xatu.DecoratedEvent) bool

	// NewBatch returns a ColumnarBatch for typed, zero-reflection
	// insertion.
	NewBatch() ColumnarBatch
}
