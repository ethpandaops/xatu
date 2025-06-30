package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRPCMetaControlIDontWantType = xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT.String()
)

type TraceRPCMetaControlIDontWant struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRPCMetaControlIDontWant(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRPCMetaControlIDontWant {
	return &TraceRPCMetaControlIDontWant{
		log:   log.WithField("event", TraceRPCMetaControlIDontWantType),
		event: event,
	}
}

func (trr *TraceRPCMetaControlIDontWant) Type() string {
	return TraceRPCMetaControlIDontWantType
}

func (trr *TraceRPCMetaControlIDontWant) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIdontwant)
	if !ok {
		return errors.New("failed to cast event data to TraceRPCMetaControlIDontWant")
	}

	return nil
}

func (trr *TraceRPCMetaControlIDontWant) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceRPCMetaControlIDontWant) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
