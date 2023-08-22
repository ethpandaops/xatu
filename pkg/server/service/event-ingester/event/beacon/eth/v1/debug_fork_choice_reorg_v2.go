package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	DebugForkChoiceReorgV2Type = "BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG_V2"
)

type DebugForkChoiceReorgV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewDebugForkChoiceReorgV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *DebugForkChoiceReorgV2 {
	return &DebugForkChoiceReorgV2{
		log:   log.WithField("event", DebugForkChoiceReorgV2Type),
		event: event,
	}
}

func (b *DebugForkChoiceReorgV2) Type() string {
	return DebugForkChoiceReorgV2Type
}

func (b *DebugForkChoiceReorgV2) Validate(_ context.Context) error {
	event, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1ForkChoiceReorgV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	if event.EthV1ForkChoiceReorgV2.Before == nil && event.EthV1ForkChoiceReorgV2.After == nil {
		return errors.New("both before and after fork choice snapshots are nil")
	}

	return nil
}

func (b *DebugForkChoiceReorgV2) Filter(_ context.Context) bool {
	return false
}
