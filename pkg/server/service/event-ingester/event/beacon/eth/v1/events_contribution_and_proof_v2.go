package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsContributionAndProofV2Type = "BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2"
)

type EventsContributionAndProofV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsContributionAndProofV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsContributionAndProofV2 {
	return &EventsContributionAndProofV2{
		log:   log.WithField("event", EventsContributionAndProofV2Type),
		event: event,
	}
}

func (b *EventsContributionAndProofV2) Type() string {
	return EventsContributionAndProofV2Type
}

func (b *EventsContributionAndProofV2) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsContributionAndProofV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsContributionAndProofV2) Filter(ctx context.Context) bool {
	return false
}
