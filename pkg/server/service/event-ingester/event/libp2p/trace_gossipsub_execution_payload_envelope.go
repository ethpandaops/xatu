package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	TraceGossipSubExecutionPayloadEnvelopeType = "LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_ENVELOPE"
)

type TraceGossipSubExecutionPayloadEnvelope struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubExecutionPayloadEnvelope(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceGossipSubExecutionPayloadEnvelope {
	return &TraceGossipSubExecutionPayloadEnvelope{
		log:   log.WithField("event", TraceGossipSubExecutionPayloadEnvelopeType),
		event: event,
	}
}

func (e *TraceGossipSubExecutionPayloadEnvelope) Type() string {
	return TraceGossipSubExecutionPayloadEnvelopeType
}

func (e *TraceGossipSubExecutionPayloadEnvelope) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_Libp2PTraceGossipsubExecutionPayloadEnvelope)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *TraceGossipSubExecutionPayloadEnvelope) Filter(_ context.Context) bool {
	return false
}

func (e *TraceGossipSubExecutionPayloadEnvelope) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
