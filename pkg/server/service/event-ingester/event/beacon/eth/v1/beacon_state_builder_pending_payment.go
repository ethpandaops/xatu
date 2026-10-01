package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	BeaconStateBuilderPendingPaymentType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_PAYMENT.String()
)

type BeaconStateBuilderPendingPayment struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconStateBuilderPendingPayment(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconStateBuilderPendingPayment {
	return &BeaconStateBuilderPendingPayment{
		log:   log.WithField("event", BeaconStateBuilderPendingPaymentType),
		event: event,
	}
}

func (b *BeaconStateBuilderPendingPayment) Type() string {
	return BeaconStateBuilderPendingPaymentType
}

func (b *BeaconStateBuilderPendingPayment) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStateBuilderPendingPayment)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconStateBuilderPendingPayment) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconStateBuilderPendingPayment) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
