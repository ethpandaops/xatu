package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconSyncCommitteeRewardType = "BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD"
)

type BeaconSyncCommitteeReward struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconSyncCommitteeReward(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconSyncCommitteeReward {
	return &BeaconSyncCommitteeReward{
		log:   log.WithField("event", BeaconSyncCommitteeRewardType),
		event: event,
	}
}

func (b *BeaconSyncCommitteeReward) Type() string {
	return BeaconSyncCommitteeRewardType
}

func (b *BeaconSyncCommitteeReward) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconSyncCommitteeReward)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconSyncCommitteeReward) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconSyncCommitteeReward) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
