package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRPCMetaControlIWantType = xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT.String()
)

type TraceRPCMetaControlIWant struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRPCMetaControlIWant(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRPCMetaControlIWant {
	return &TraceRPCMetaControlIWant{
		log:   log.WithField("event", TraceRPCMetaControlIWantType),
		event: event,
	}
}

func (trr *TraceRPCMetaControlIWant) Type() string {
	return TraceRPCMetaControlIWantType
}

func (trr *TraceRPCMetaControlIWant) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIwant)
	if !ok {
		return errors.New("failed to cast event data to TraceRPCMetaControlIWant")
	}

	return nil
}

func (trr *TraceRPCMetaControlIWant) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceRPCMetaControlIWant) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
