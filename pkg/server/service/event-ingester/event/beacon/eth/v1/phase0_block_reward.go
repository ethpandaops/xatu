package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconPhase0BlockRewardType = "BEACON_API_ETH_V1_BEACON_PHASE0_BLOCK_REWARD"
)

type BeaconPhase0BlockReward struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconPhase0BlockReward(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconPhase0BlockReward {
	return &BeaconPhase0BlockReward{
		log:   log.WithField("event", BeaconPhase0BlockRewardType),
		event: event,
	}
}

func (b *BeaconPhase0BlockReward) Type() string {
	return BeaconPhase0BlockRewardType
}

func (b *BeaconPhase0BlockReward) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconPhase0BlockReward)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconPhase0BlockReward) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconPhase0BlockReward) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
