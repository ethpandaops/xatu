package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceConnectedType = xatu.Event_LIBP2P_TRACE_CONNECTED.String()
)

type TraceConnected struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceConnected(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceConnected {
	return &TraceConnected{
		log:   log.WithField("event", TraceConnectedType),
		event: event,
	}
}

func (tc *TraceConnected) Type() string {
	return TraceConnectedType
}

func (tc *TraceConnected) Validate(ctx context.Context) error {
	_, ok := tc.event.Data.(*xatu.DecoratedEvent_Libp2PTraceConnected)
	if !ok {
		return errors.New("failed to cast event data to TraceConnected")
	}

	return nil
}

func (tc *TraceConnected) Filter(ctx context.Context) bool {
	return false
}

func (tc *TraceConnected) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
