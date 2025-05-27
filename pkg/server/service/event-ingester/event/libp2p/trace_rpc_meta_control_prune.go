package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRPCMetaControlPruneType = xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE.String()
)

type TraceRPCMetaControlPrune struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRPCMetaControlPrune(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRPCMetaControlPrune {
	return &TraceRPCMetaControlPrune{
		log:   log.WithField("event", TraceRPCMetaControlPruneType),
		event: event,
	}
}

func (trr *TraceRPCMetaControlPrune) Type() string {
	return TraceRPCMetaControlPruneType
}

func (trr *TraceRPCMetaControlPrune) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRpcMetaControlPrune)
	if !ok {
		return errors.New("failed to cast event data to TraceRPCMetaControlPrune")
	}

	return nil
}

func (trr *TraceRPCMetaControlPrune) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceRPCMetaControlPrune) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
