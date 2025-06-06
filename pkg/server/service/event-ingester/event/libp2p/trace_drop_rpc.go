package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceDropRPCType = xatu.Event_LIBP2P_TRACE_DROP_RPC.String()
)

type TraceDropRPC struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceDropRPC(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceDropRPC {
	return &TraceDropRPC{
		log:   log.WithField("event", TraceDropRPCType),
		event: event,
	}
}

func (trr *TraceDropRPC) Type() string {
	return TraceDropRPCType
}

func (trr *TraceDropRPC) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDropRpc)
	if !ok {
		return errors.New("failed to cast event data to TraceDropRPC")
	}

	return nil
}

func (trr *TraceDropRPC) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceDropRPC) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
