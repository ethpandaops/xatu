package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRejectMessageType = xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE.String()
)

type TraceRejectMessage struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRejectMessage(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRejectMessage {
	return &TraceRejectMessage{
		log:   log.WithField("event", TraceRejectMessageType),
		event: event,
	}
}

func (trm *TraceRejectMessage) Type() string {
	return TraceRejectMessageType
}

func (trm *TraceRejectMessage) Validate(ctx context.Context) error {
	_, ok := trm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRejectMessage)
	if !ok {
		return errors.New("failed to cast event data to TraceRejectMessage")
	}

	return nil
}

func (trm *TraceRejectMessage) Filter(ctx context.Context) bool {
	return false
}

func (trm *TraceRejectMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
