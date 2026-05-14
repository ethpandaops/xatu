package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceJoinType = xatu.Event_LIBP2P_TRACE_JOIN.String()
)

type TraceJoin struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceJoin(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceJoin {
	return &TraceJoin{
		log:   log.WithField("event", TraceJoinType),
		event: event,
	}
}

func (tjm *TraceJoin) Type() string {
	return TraceJoinType
}

func (tjm *TraceJoin) Validate(ctx context.Context) error {
	_, ok := tjm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceJoin)
	if !ok {
		return errors.New("failed to cast event data to TraceJoin")
	}

	return nil
}

func (tjm *TraceJoin) Filter(ctx context.Context) bool {
	return false
}

func (tjm *TraceJoin) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
