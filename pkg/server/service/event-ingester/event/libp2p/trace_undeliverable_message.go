package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceUndeliverableMessageType = xatu.Event_LIBP2P_TRACE_UNDELIVERABLE_MESSAGE.String()
)

type TraceUndeliverableMessage struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceUndeliverableMessage(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceUndeliverableMessage {
	return &TraceUndeliverableMessage{
		log:   log.WithField("event", TraceUndeliverableMessageType),
		event: event,
	}
}

func (tum *TraceUndeliverableMessage) Type() string {
	return TraceUndeliverableMessageType
}

func (tum *TraceUndeliverableMessage) Validate(ctx context.Context) error {
	_, ok := tum.event.Data.(*xatu.DecoratedEvent_Libp2PTraceUndeliverableMessage)
	if !ok {
		return errors.New("failed to cast event data to TraceUndeliverableMessage")
	}

	return nil
}

func (tum *TraceUndeliverableMessage) Filter(ctx context.Context) bool {
	return false
}

func (tum *TraceUndeliverableMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
