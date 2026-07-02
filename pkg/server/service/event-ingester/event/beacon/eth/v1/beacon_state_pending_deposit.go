package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	BeaconStatePendingDepositType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT.String()
)

type BeaconStatePendingDeposit struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconStatePendingDeposit(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconStatePendingDeposit {
	return &BeaconStatePendingDeposit{
		log:   log.WithField("event", BeaconStatePendingDepositType),
		event: event,
	}
}

func (b *BeaconStatePendingDeposit) Type() string {
	return BeaconStatePendingDepositType
}

func (b *BeaconStatePendingDeposit) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStatePendingDeposit)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconStatePendingDeposit) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconStatePendingDeposit) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
