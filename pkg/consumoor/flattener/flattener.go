package flattener

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Flattener converts a DecoratedEvent into flat ClickHouse rows for a
// specific target table. Each implementation handles one or more event
// names and produces rows for exactly one ClickHouse table.
type Flattener interface {
	// EventNames returns the event names this flattener handles.
	EventNames() []xatu.Event_Name

	// TableName returns the target ClickHouse table name.
	TableName() string

	// Flatten converts a DecoratedEvent into one or more flat rows
	// suitable for ClickHouse insertion. Each row is a map of column
	// name to value. Returns multiple rows for fan-out cases.
	Flatten(event *xatu.DecoratedEvent, meta *metadata.CommonMetadata) ([]map[string]any, error)

	// ShouldProcess returns whether this event should be processed by
	// this flattener. Used for conditional routing where the same event
	// name routes to different tables based on additional_data fields.
	ShouldProcess(event *xatu.DecoratedEvent) bool
}
