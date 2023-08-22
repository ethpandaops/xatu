package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	DebugForkChoiceReorgType = "BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG"
)

type DebugForkChoiceReorg struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewDebugForkChoiceReorg(log logrus.FieldLogger, event *xatu.DecoratedEvent) *DebugForkChoiceReorg {
	return &DebugForkChoiceReorg{
		log:   log.WithField("event", DebugForkChoiceReorgType),
		event: event,
	}
}

func (b *DebugForkChoiceReorg) Type() string {
	return DebugForkChoiceReorgType
}

func (b *DebugForkChoiceReorg) Validate(_ context.Context) error {
	event, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1ForkChoiceReorg)
	if !ok {
		return errors.New("failed to cast event data")
	}

	if event.EthV1ForkChoiceReorg.Before == nil && event.EthV1ForkChoiceReorg.After == nil {
		return errors.New("both before and after fork choice snapshots are nil")
	}

	return nil
}

func (b *DebugForkChoiceReorg) Filter(_ context.Context) bool {
	return false
}
