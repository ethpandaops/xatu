package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsFinalizedCheckpointV2Type = "BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2"
)

type EventsFinalizedCheckpointV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsFinalizedCheckpointV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsFinalizedCheckpointV2 {
	return &EventsFinalizedCheckpointV2{
		log:   log.WithField("event", EventsFinalizedCheckpointV2Type),
		event: event,
	}
}

func (b *EventsFinalizedCheckpointV2) Type() string {
	return EventsFinalizedCheckpointV2Type
}

func (b *EventsFinalizedCheckpointV2) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsFinalizedCheckpointV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsFinalizedCheckpointV2) Filter(ctx context.Context) bool {
	return false
}
