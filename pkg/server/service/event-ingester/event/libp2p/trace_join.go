package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceJoinType = xatu.Event_LIBP2P_TRACE_JOIN.String()
)

type TraceJoin struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceJoin(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceJoin {
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
