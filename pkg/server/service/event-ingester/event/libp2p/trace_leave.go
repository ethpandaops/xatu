package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceLeaveType = xatu.Event_LIBP2P_TRACE_LEAVE.String()
)

type TraceLeave struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceLeave(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceLeave {
	return &TraceLeave{
		log:   log.WithField("event", TraceLeaveType),
		event: event,
	}
}

func (tlm *TraceLeave) Type() string {
	return TraceLeaveType
}

func (tlm *TraceLeave) Validate(ctx context.Context) error {
	_, ok := tlm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceLeave)
	if !ok {
		return errors.New("failed to cast event data to TraceLeave")
	}

	return nil
}

func (tlm *TraceLeave) Filter(ctx context.Context) bool {
	return false
}

func (tlm *TraceLeave) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
