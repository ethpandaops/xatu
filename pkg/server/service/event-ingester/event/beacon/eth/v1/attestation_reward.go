package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconAttestationRewardType = "BEACON_API_ETH_V1_BEACON_ATTESTATION_REWARD"
)

type BeaconAttestationReward struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconAttestationReward(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconAttestationReward {
	return &BeaconAttestationReward{
		log:   log.WithField("event", BeaconAttestationRewardType),
		event: event,
	}
}

func (b *BeaconAttestationReward) Type() string {
	return BeaconAttestationRewardType
}

func (b *BeaconAttestationReward) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconAttestationReward)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconAttestationReward) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconAttestationReward) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
