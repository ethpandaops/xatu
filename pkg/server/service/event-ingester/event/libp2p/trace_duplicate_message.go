package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceDuplicateMessageType = xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String()
)

type TraceDuplicateMessage struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceDuplicateMessage(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceDuplicateMessage {
	return &TraceDuplicateMessage{
		log:   log.WithField("event", TraceDuplicateMessageType),
		event: event,
	}
}

func (tdm *TraceDuplicateMessage) Type() string {
	return TraceDuplicateMessageType
}

func (tdm *TraceDuplicateMessage) Validate(ctx context.Context) error {
	_, ok := tdm.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDuplicateMessage)
	if !ok {
		return errors.New("failed to cast event data to TraceDuplicateMessage")
	}

	return nil
}

func (tdm *TraceDuplicateMessage) Filter(ctx context.Context) bool {
	return false
}

func (tdm *TraceDuplicateMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
