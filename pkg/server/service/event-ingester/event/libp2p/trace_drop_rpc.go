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

func (tdr *TraceDropRPC) Type() string {
	return TraceDropRPCType
}

func (tdr *TraceDropRPC) Validate(ctx context.Context) error {
	_, ok := tdr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDropRpc)
	if !ok {
		return errors.New("failed to cast event data to TraceDropRPC")
	}

	return nil
}

func (tdr *TraceDropRPC) Filter(ctx context.Context) bool {
	return false
}

func (tdr *TraceDropRPC) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
