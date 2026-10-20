package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceGossipSubPartialDataColumnType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN.String()
)

type TraceGossipSubPartialDataColumn struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubPartialDataColumn(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceGossipSubPartialDataColumn {
	return &TraceGossipSubPartialDataColumn{
		log:   log.WithField("event", TraceGossipSubPartialDataColumnType),
		event: event,
	}
}

// AppendServerMeta implements event.Event.
func (t *TraceGossipSubPartialDataColumn) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}

// Filter implements event.Event.
func (t *TraceGossipSubPartialDataColumn) Filter(ctx context.Context) bool {
	return false
}

// Type implements event.Event.
func (t *TraceGossipSubPartialDataColumn) Type() string {
	return TraceGossipSubPartialDataColumnType
}

// Validate implements event.Event.
func (t *TraceGossipSubPartialDataColumn) Validate(ctx context.Context) error {
	if t.event.GetLibp2PTraceGossipsubPartialDataColumn() == nil {
		return errors.New("failed to cast event data to TraceGossipSubPartialDataColumn")
	}

	return nil
}
