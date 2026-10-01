package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	BeaconStateBuilderPendingWithdrawalType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_WITHDRAWAL.String()
)

type BeaconStateBuilderPendingWithdrawal struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconStateBuilderPendingWithdrawal(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconStateBuilderPendingWithdrawal {
	return &BeaconStateBuilderPendingWithdrawal{
		log:   log.WithField("event", BeaconStateBuilderPendingWithdrawalType),
		event: event,
	}
}

func (b *BeaconStateBuilderPendingWithdrawal) Type() string {
	return BeaconStateBuilderPendingWithdrawalType
}

func (b *BeaconStateBuilderPendingWithdrawal) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStateBuilderPendingWithdrawal)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconStateBuilderPendingWithdrawal) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconStateBuilderPendingWithdrawal) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
