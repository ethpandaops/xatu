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

func (tlm *TraceRejectMessage) Type() string {
	return TraceRejectMessageType
}

func (tlm *TraceRejectMessage) Validate(ctx context.Context) error {
	if _, ok := tlm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRejectMessage); !ok {
		return errors.New("failed to cast event data to TraceRejectMessage")
	}

	return nil
}

func (tlm *TraceRejectMessage) Filter(ctx context.Context) bool {
	return false
}

func (tlm *TraceRejectMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
