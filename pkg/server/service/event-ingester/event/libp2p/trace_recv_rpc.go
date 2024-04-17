package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRecvRPCType = xatu.Event_LIBP2P_TRACE_RECV_RPC.String()
)

type TraceRecvRPC struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRecvRPC(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRecvRPC {
	return &TraceRecvRPC{
		log:   log.WithField("event", TraceRecvRPCType),
		event: event,
	}
}

func (trr *TraceRecvRPC) Type() string {
	return TraceRecvRPCType
}

func (trr *TraceRecvRPC) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRecvRpc)
	if !ok {
		return errors.New("failed to cast event data to TraceRecvRPC")
	}

	return nil
}

func (trr *TraceRecvRPC) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceRecvRPC) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
