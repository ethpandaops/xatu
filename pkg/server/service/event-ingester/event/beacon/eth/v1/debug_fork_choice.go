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
	log       logrus.FieldLogger
	event     *xatu.DecoratedEvent
	networkID uint64
}

func NewDebugForkChoice(log logrus.FieldLogger, event *xatu.DecoratedEvent, networkID uint64) *DebugForkChoice {
	return &DebugForkChoice{
		log:       log.WithField("event", DebugForkChoiceType),
		event:     event,
		networkID: networkID,
	}
}

func (b *DebugForkChoice) Type() string {
	return EventsHeadType
}

func (b *DebugForkChoice) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1ForkChoice)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *DebugForkChoice) Filter(_ context.Context) bool {
	networkID := b.event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()

	return networkID != b.networkID
}
