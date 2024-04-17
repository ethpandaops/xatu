package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceSendRPCType = xatu.Event_LIBP2P_TRACE_SEND_RPC.String()
)

type TraceSendRPC struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceSendRPC(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceSendRPC {
	return &TraceSendRPC{
		log:   log.WithField("event", TraceSendRPCType),
		event: event,
	}
}

func (trr *TraceSendRPC) Type() string {
	return TraceSendRPCType
}

func (trr *TraceSendRPC) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceSendRpc)
	if !ok {
		return errors.New("failed to cast event data to TraceSendRPC")
	}

	return nil
}

func (trr *TraceSendRPC) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceSendRPC) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
