package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceHandleStatusType = xatu.Event_LIBP2P_TRACE_HANDLE_STATUS.String()
)

type TraceHandleStatus struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceHandleStatus(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceHandleStatus {
	return &TraceHandleStatus{
		log:   log.WithField("event", TraceHandleStatusType),
		event: event,
	}
}

func (trr *TraceHandleStatus) Type() string {
	return TraceSendRPCType
}

func (trr *TraceHandleStatus) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceHandleStatus)
	if !ok {
		return errors.New("failed to cast event data to TraceHandleStatus")
	}

	return nil
}

func (trr *TraceHandleStatus) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceHandleStatus) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
