package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	BeaconStatePendingConsolidationType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION.String()
)

type BeaconStatePendingConsolidation struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconStatePendingConsolidation(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconStatePendingConsolidation {
	return &BeaconStatePendingConsolidation{
		log:   log.WithField("event", BeaconStatePendingConsolidationType),
		event: event,
	}
}

func (b *BeaconStatePendingConsolidation) Type() string {
	return BeaconStatePendingConsolidationType
}

func (b *BeaconStatePendingConsolidation) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStatePendingConsolidation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconStatePendingConsolidation) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconStatePendingConsolidation) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
