package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRemovePeerType = xatu.Event_LIBP2P_TRACE_REMOVE_PEER.String()
)

type TraceRemovePeer struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRemovePeer(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRemovePeer {
	return &TraceRemovePeer{
		log:   log.WithField("event", TraceRemovePeerType),
		event: event,
	}
}

func (trp *TraceRemovePeer) Type() string {
	return TraceRemovePeerType
}

func (trp *TraceRemovePeer) Validate(ctx context.Context) error {
	_, ok := trp.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRemovePeer)
	if !ok {
		return errors.New("failed to cast event data to TraceRemovePeer")
	}

	return nil
}

func (trp *TraceRemovePeer) Filter(ctx context.Context) bool {
	return false
}

func (trp *TraceRemovePeer) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
