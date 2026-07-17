package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceGossipSubMessagePayloadType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_MESSAGE_PAYLOAD.String()
)

type TraceGossipSubMessagePayload struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubMessagePayload(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceGossipSubMessagePayload {
	return &TraceGossipSubMessagePayload{
		log:   log.WithField("event", TraceGossipSubMessagePayloadType),
		event: event,
	}
}

// AppendServerMeta implements event.Event.
func (t *TraceGossipSubMessagePayload) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}

// Filter implements event.Event.
func (t *TraceGossipSubMessagePayload) Filter(ctx context.Context) bool {
	return false
}

// Type implements event.Event.
func (t *TraceGossipSubMessagePayload) Type() string {
	return TraceGossipSubMessagePayloadType
}

// Validate implements event.Event.
func (t *TraceGossipSubMessagePayload) Validate(ctx context.Context) error {
	if t.event.GetLibp2PTraceGossipsubMessagePayload() == nil {
		return errors.New("failed to cast event data to TraceGossipSubMessagePayload")
	}

	return nil
}
