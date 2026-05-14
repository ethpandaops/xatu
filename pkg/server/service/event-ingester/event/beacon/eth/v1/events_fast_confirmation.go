package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsFastConfirmationType = "BEACON_API_ETH_V1_EVENTS_FAST_CONFIRMATION"
)

type EventsFastConfirmation struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsFastConfirmation(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsFastConfirmation {
	return &EventsFastConfirmation{
		log:   log.WithField("event", EventsFastConfirmationType),
		event: event,
	}
}

func (b *EventsFastConfirmation) Type() string {
	return EventsFastConfirmationType
}

func (b *EventsFastConfirmation) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsFastConfirmation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsFastConfirmation) Filter(ctx context.Context) bool {
	return false
}

func (b *EventsFastConfirmation) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
