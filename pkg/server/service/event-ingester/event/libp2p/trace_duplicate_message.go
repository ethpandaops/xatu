package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceDuplicateMessageType = xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String()
)

type TraceDuplicateMessage struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceDuplicateMessage(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceDuplicateMessage {
	return &TraceDuplicateMessage{
		log:   log.WithField("event", TraceDuplicateMessageType),
		event: event,
	}
}

func (tlm *TraceDuplicateMessage) Type() string {
	return TraceDuplicateMessageType
}

func (tlm *TraceDuplicateMessage) Validate(ctx context.Context) error {
	if _, ok := tlm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDuplicateMessage); !ok {
		return errors.New("failed to cast event data to TraceDuplicateMessage")
	}

	return nil
}

func (tlm *TraceDuplicateMessage) Filter(ctx context.Context) bool {
	return false
}

func (tlm *TraceDuplicateMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
