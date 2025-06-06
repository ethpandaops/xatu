package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRPCMetaControlGraftType = xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT.String()
)

type TraceRPCMetaControlGraft struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRPCMetaControlGraft(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRPCMetaControlGraft {
	return &TraceRPCMetaControlGraft{
		log:   log.WithField("event", TraceRPCMetaControlGraftType),
		event: event,
	}
}

func (trr *TraceRPCMetaControlGraft) Type() string {
	return TraceRPCMetaControlGraftType
}

func (trr *TraceRPCMetaControlGraft) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRpcMetaControlGraft)
	if !ok {
		return errors.New("failed to cast event data to TraceRPCMetaControlGraft")
	}

	return nil
}

func (trr *TraceRPCMetaControlGraft) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceRPCMetaControlGraft) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
