package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceGossipSubPartialDataColumnOutcomeType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME.String()
)

type TraceGossipSubPartialDataColumnOutcome struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubPartialDataColumnOutcome(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceGossipSubPartialDataColumnOutcome {
	return &TraceGossipSubPartialDataColumnOutcome{
		log:   log.WithField("event", TraceGossipSubPartialDataColumnOutcomeType),
		event: event,
	}
}

// AppendServerMeta implements event.Event.
func (t *TraceGossipSubPartialDataColumnOutcome) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}

// Filter implements event.Event.
func (t *TraceGossipSubPartialDataColumnOutcome) Filter(ctx context.Context) bool {
	return false
}

// Type implements event.Event.
func (t *TraceGossipSubPartialDataColumnOutcome) Type() string {
	return TraceGossipSubPartialDataColumnOutcomeType
}

// Validate implements event.Event.
func (t *TraceGossipSubPartialDataColumnOutcome) Validate(ctx context.Context) error {
	if t.event.GetLibp2PTraceGossipsubPartialDataColumnOutcome() == nil {
		return errors.New("failed to cast event data to TraceGossipSubPartialDataColumnOutcome")
	}

	return nil
}
