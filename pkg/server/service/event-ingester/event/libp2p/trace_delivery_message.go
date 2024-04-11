package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceDeliverMessageType = xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE.String()
)

type TraceDeliverMessage struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceDeliverMessage(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceDeliverMessage {
	return &TraceDeliverMessage{
		log:   log.WithField("event", TraceDeliverMessageType),
		event: event,
	}
}

func (tdm *TraceDeliverMessage) Type() string {
	return TraceDeliverMessageType
}

func (tdm *TraceDeliverMessage) Validate(ctx context.Context) error {
	_, ok := tdm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDeliverMessage)
	if !ok {
		return errors.New("failed to cast event data to TraceDeliverMessage")
	}

	return nil
}

func (tdm *TraceDeliverMessage) Filter(ctx context.Context) bool {
	return false
}

func (tdm *TraceDeliverMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
