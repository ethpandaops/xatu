package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsBlobSidecarType = "BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR"
)

type EventsBlobSidecar struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsBlobSidecar(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsBlobSidecar {
	return &EventsBlobSidecar{
		log:   log.WithField("event", EventsBlobSidecarType),
		event: event,
	}
}

func (b *EventsBlobSidecar) Type() string {
	return EventsBlobSidecarType
}

func (b *EventsBlobSidecar) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsBlobSidecar)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsBlobSidecar) Filter(ctx context.Context) bool {
	return false
}
