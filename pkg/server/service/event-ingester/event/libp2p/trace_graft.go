package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceGraftType = xatu.Event_LIBP2P_TRACE_GRAFT.String()
)

type TraceGraft struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceGraft(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceGraft {
	return &TraceGraft{
		log:   log.WithField("event", TraceGraftType),
		event: event,
	}
}

func (tgm *TraceGraft) Type() string {
	return TraceGraftType
}

func (tgm *TraceGraft) Validate(ctx context.Context) error {
	_, ok := tgm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceGraft)
	if !ok {
		return errors.New("failed to cast event data to TraceGraft")
	}

	return nil
}

func (tgm *TraceGraft) Filter(ctx context.Context) bool {
	return false
}

func (tgm *TraceGraft) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
