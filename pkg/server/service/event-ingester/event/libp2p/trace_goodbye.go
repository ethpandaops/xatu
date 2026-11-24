package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceGoodbyeType = xatu.Event_LIBP2P_TRACE_GOODBYE.String()
)

type TraceGoodbye struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceGoodbye(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceGoodbye {
	return &TraceGoodbye{
		log:   log.WithField("event", TraceGoodbyeType),
		event: event,
	}
}

func (t *TraceGoodbye) Type() string {
	return TraceGoodbyeType
}

func (t *TraceGoodbye) Validate(ctx context.Context) error {
	if _, ok := t.event.GetData().(*xatu.DecoratedEvent_Libp2PTraceGoodbye); !ok {
		return errors.New("failed to cast event data to TraceGoodbye")
	}

	return nil
}

func (t *TraceGoodbye) Filter(ctx context.Context) bool {
	return false
}

func (t *TraceGoodbye) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
