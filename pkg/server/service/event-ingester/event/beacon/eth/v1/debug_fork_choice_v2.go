package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	DebugForkChoiceV2Type = "BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_V2"
)

type DebugForkChoiceV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewDebugForkChoiceV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *DebugForkChoiceV2 {
	return &DebugForkChoiceV2{
		log:   log.WithField("event", DebugForkChoiceV2Type),
		event: event,
	}
}

func (b *DebugForkChoiceV2) Type() string {
	return DebugForkChoiceV2Type
}

func (b *DebugForkChoiceV2) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1ForkChoiceV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *DebugForkChoiceV2) Filter(_ context.Context) bool {
	return false
}

func (b *DebugForkChoiceV2) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
