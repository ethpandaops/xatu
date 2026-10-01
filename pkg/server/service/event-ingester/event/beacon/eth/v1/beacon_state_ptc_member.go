package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	BeaconStatePtcMemberType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PTC_MEMBER.String()
)

type BeaconStatePtcMember struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconStatePtcMember(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconStatePtcMember {
	return &BeaconStatePtcMember{
		log:   log.WithField("event", BeaconStatePtcMemberType),
		event: event,
	}
}

func (b *BeaconStatePtcMember) Type() string {
	return BeaconStatePtcMemberType
}

func (b *BeaconStatePtcMember) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStatePtcMember)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconStatePtcMember) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconStatePtcMember) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
