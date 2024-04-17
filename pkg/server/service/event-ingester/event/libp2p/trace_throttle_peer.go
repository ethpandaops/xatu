package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceThrottlePeerType = xatu.Event_LIBP2P_TRACE_THROTTLE_PEER.String()
)

type TraceThrottlePeer struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceThrottlePeer(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceThrottlePeer {
	return &TraceThrottlePeer{
		log:   log.WithField("event", TraceThrottlePeerType),
		event: event,
	}
}

func (ttp *TraceThrottlePeer) Type() string {
	return TraceThrottlePeerType
}

func (ttp *TraceThrottlePeer) Validate(ctx context.Context) error {
	_, ok := ttp.event.Data.(*xatu.DecoratedEvent_Libp2PTraceThrottlePeer)
	if !ok {
		return errors.New("failed to cast event data to TraceThrottlePeer")
	}

	return nil
}

func (ttp *TraceThrottlePeer) Filter(ctx context.Context) bool {
	return false
}

func (ttp *TraceThrottlePeer) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
