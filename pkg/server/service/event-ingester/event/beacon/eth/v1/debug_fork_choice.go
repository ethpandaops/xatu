package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	DebugForkChoiceType = "BEACON_API_ETH_V1_DEBUG_FORK_CHOICE"
)

type DebugForkChoice struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewDebugForkChoice(log logrus.FieldLogger, event *xatu.DecoratedEvent) *DebugForkChoice {
	return &DebugForkChoice{
		log:   log.WithField("event", DebugForkChoiceType),
		event: event,
	}
}

func (b *DebugForkChoice) Type() string {
	return EventsHeadType
}

func (b *DebugForkChoice) Validate(_ context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV1ForkChoice)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *DebugForkChoice) Filter(_ context.Context) bool {
	return false
}
