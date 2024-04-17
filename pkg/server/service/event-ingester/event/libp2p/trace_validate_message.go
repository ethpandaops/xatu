package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceValidateMessageType = xatu.Event_LIBP2P_TRACE_VALIDATE_MESSAGE.String()
)

type TraceValidateMessage struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceValidateMessage(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceValidateMessage {
	return &TraceValidateMessage{
		log:   log.WithField("event", TraceValidateMessageType),
		event: event,
	}
}

func (tvm *TraceValidateMessage) Type() string {
	return TraceValidateMessageType
}

func (tvm *TraceValidateMessage) Validate(ctx context.Context) error {
	_, ok := tvm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceValidateMessage)
	if !ok {
		return errors.New("failed to cast event data to TraceValidateMessage")
	}

	return nil
}

func (tvm *TraceValidateMessage) Filter(ctx context.Context) bool {
	return false
}

func (tvm *TraceValidateMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
