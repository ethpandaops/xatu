package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BlockRewardType = "BEACON_API_ETH_V1_BEACON_BLOCK_REWARD"
)

type BlockReward struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBlockReward(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BlockReward {
	return &BlockReward{
		log:   log.WithField("event", BlockRewardType),
		event: event,
	}
}

func (b *BlockReward) Type() string {
	return BlockRewardType
}

func (b *BlockReward) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconBlockReward)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BlockReward) Filter(_ context.Context) bool {
	return false
}

func (b *BlockReward) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
