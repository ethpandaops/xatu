package flattener

import "errors"

// ErrInvalidEvent is returned by FlattenTo when an event has nil proto
// wrapper fields that map to non-nullable ClickHouse columns. The
// table writer skips these events individually without failing the
// entire batch.
var ErrInvalidEvent = errors.New("invalid event")
