package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRPCMetaMessageType = xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE.String()
)

type TraceRPCMetaMessage struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRPCMetaMessage(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRPCMetaMessage {
	return &TraceRPCMetaMessage{
		log:   log.WithField("event", TraceRPCMetaMessageType),
		event: event,
	}
}

func (trr *TraceRPCMetaMessage) Type() string {
	return TraceRPCMetaMessageType
}

func (trr *TraceRPCMetaMessage) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRpcMetaMessage)
	if !ok {
		return errors.New("failed to cast event data to TraceRPCMetaMessage")
	}

	return nil
}

func (trr *TraceRPCMetaMessage) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceRPCMetaMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
