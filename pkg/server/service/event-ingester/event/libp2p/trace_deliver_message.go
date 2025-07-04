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

func (tlm *TraceDeliverMessage) Type() string {
	return TraceDeliverMessageType
}

func (tlm *TraceDeliverMessage) Validate(ctx context.Context) error {
	if _, ok := tlm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDeliverMessage); !ok {
		return errors.New("failed to cast event data to TraceDeliverMessage")
	}

	return nil
}

func (tlm *TraceDeliverMessage) Filter(ctx context.Context) bool {
	return false
}

func (tlm *TraceDeliverMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
