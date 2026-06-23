package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	BeaconStatePendingPartialWithdrawalType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL.String()
)

type BeaconStatePendingPartialWithdrawal struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconStatePendingPartialWithdrawal(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconStatePendingPartialWithdrawal {
	return &BeaconStatePendingPartialWithdrawal{
		log:   log.WithField("event", BeaconStatePendingPartialWithdrawalType),
		event: event,
	}
}

func (b *BeaconStatePendingPartialWithdrawal) Type() string {
	return BeaconStatePendingPartialWithdrawalType
}

func (b *BeaconStatePendingPartialWithdrawal) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStatePendingPartialWithdrawal)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconStatePendingPartialWithdrawal) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconStatePendingPartialWithdrawal) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
