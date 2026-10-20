package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceDataColumnCompletedType = xatu.Event_LIBP2P_TRACE_DATA_COLUMN_COMPLETED.String()
)

type TraceDataColumnCompleted struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceDataColumnCompleted(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceDataColumnCompleted {
	return &TraceDataColumnCompleted{
		log:   log.WithField("event", TraceDataColumnCompletedType),
		event: event,
	}
}

// AppendServerMeta implements event.Event.
func (t *TraceDataColumnCompleted) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}

// Filter implements event.Event.
func (t *TraceDataColumnCompleted) Filter(ctx context.Context) bool {
	return false
}

// Type implements event.Event.
func (t *TraceDataColumnCompleted) Type() string {
	return TraceDataColumnCompletedType
}

// Validate implements event.Event.
func (t *TraceDataColumnCompleted) Validate(ctx context.Context) error {
	if t.event.GetLibp2PTraceDataColumnCompleted() == nil {
		return errors.New("failed to cast event data to TraceDataColumnCompleted")
	}

	return nil
}
