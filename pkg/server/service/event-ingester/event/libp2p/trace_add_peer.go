package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceAddPeerType = xatu.Event_LIBP2P_TRACE_ADD_PEER.String()
)

type TraceAddPeer struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceAddPeer(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceAddPeer {
	return &TraceAddPeer{
		log:   log.WithField("event", TraceAddPeerType),
		event: event,
	}
}

func (tap *TraceAddPeer) Type() string {
	return TraceAddPeerType
}

func (tap *TraceAddPeer) Validate(ctx context.Context) error {
	_, ok := tap.event.Data.(*xatu.DecoratedEvent_Libp2PTraceAddPeer)
	if !ok {
		return errors.New("failed to cast event data to TraceAddPeer")
	}

	return nil
}

func (tap *TraceAddPeer) Filter(ctx context.Context) bool {
	return false
}

func (tap *TraceAddPeer) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
