package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsFinalizedCheckpointType = "BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT"
)

type EventsFinalizedCheckpoint struct {
	log       logrus.FieldLogger
	event     *xatu.DecoratedEvent
	networkID uint64
}

func NewEventsFinalizedCheckpoint(log logrus.FieldLogger, event *xatu.DecoratedEvent, networkID uint64) *EventsFinalizedCheckpoint {
	return &EventsFinalizedCheckpoint{
		log:       log.WithField("event", EventsFinalizedCheckpointType),
		event:     event,
		networkID: networkID,
	}
}

func (b *EventsFinalizedCheckpoint) Type() string {
	return EventsFinalizedCheckpointType
}

func (b *EventsFinalizedCheckpoint) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsFinalizedCheckpoint)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsFinalizedCheckpoint) Filter(ctx context.Context) bool {
	networkID := b.event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()

	return networkID != b.networkID
}
