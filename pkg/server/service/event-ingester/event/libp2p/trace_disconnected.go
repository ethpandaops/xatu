package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceDisconnectedType = xatu.Event_LIBP2P_TRACE_DISCONNECTED.String()
)

type TraceDisconnected struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceDisconnected(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceDisconnected {
	return &TraceDisconnected{
		log:   log.WithField("event", TraceDisconnectedType),
		event: event,
	}
}

func (td *TraceDisconnected) Type() string {
	return TraceDisconnectedType
}

func (td *TraceDisconnected) Validate(ctx context.Context) error {
	_, ok := td.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDisconnected)
	if !ok {
		return errors.New("failed to cast event data to TraceDisconnected")
	}

	return nil
}

func (td *TraceDisconnected) Filter(ctx context.Context) bool {
	return false
}

func (td *TraceDisconnected) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
