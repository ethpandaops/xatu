package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconPhase0AttestationRewardType = "BEACON_API_ETH_V1_BEACON_PHASE0_ATTESTATION_REWARD"
)

type BeaconPhase0AttestationReward struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconPhase0AttestationReward(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconPhase0AttestationReward {
	return &BeaconPhase0AttestationReward{
		log:   log.WithField("event", BeaconPhase0AttestationRewardType),
		event: event,
	}
}

func (b *BeaconPhase0AttestationReward) Type() string {
	return BeaconPhase0AttestationRewardType
}

func (b *BeaconPhase0AttestationReward) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconPhase0AttestationReward)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconPhase0AttestationReward) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconPhase0AttestationReward) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
