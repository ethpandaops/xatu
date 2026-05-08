package route

import "errors"

// ErrInvalidEvent is returned by FlattenTo when an event's payload is
// fundamentally invalid (e.g. nil protobuf message). Events flagged with
// this error are permanently unflattenable and should be dropped rather
// than retried.
var ErrInvalidEvent = errors.New("invalid event")
