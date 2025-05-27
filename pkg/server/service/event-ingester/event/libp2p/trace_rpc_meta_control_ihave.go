package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRPCMetaControlIHaveType = xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE.String()
)

type TraceRPCMetaControlIHave struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRPCMetaControlIHave(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRPCMetaControlIHave {
	return &TraceRPCMetaControlIHave{
		log:   log.WithField("event", TraceRPCMetaControlIHaveType),
		event: event,
	}
}

func (trr *TraceRPCMetaControlIHave) Type() string {
	return TraceRPCMetaControlIHaveType
}

func (trr *TraceRPCMetaControlIHave) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIhave)
	if !ok {
		return errors.New("failed to cast event data to TraceRPCMetaControlIHave")
	}

	return nil
}

func (trr *TraceRPCMetaControlIHave) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceRPCMetaControlIHave) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
