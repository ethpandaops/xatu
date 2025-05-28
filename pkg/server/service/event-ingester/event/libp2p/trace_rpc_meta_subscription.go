package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceRPCMetaSubscriptionType = xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION.String()
)

type TraceRPCMetaSubscription struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceRPCMetaSubscription(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceRPCMetaSubscription {
	return &TraceRPCMetaSubscription{
		log:   log.WithField("event", TraceRPCMetaSubscriptionType),
		event: event,
	}
}

func (trr *TraceRPCMetaSubscription) Type() string {
	return TraceRPCMetaSubscriptionType
}

func (trr *TraceRPCMetaSubscription) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceRpcMetaSubscription)
	if !ok {
		return errors.New("failed to cast event data to TraceRPCMetaSubscription")
	}

	return nil
}

func (trr *TraceRPCMetaSubscription) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceRPCMetaSubscription) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
