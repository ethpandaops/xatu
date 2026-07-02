package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconBlockExecutionRequestConsolidationType = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_CONSOLIDATION"
)

type BeaconBlockExecutionRequestConsolidation struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockExecutionRequestConsolidation(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconBlockExecutionRequestConsolidation {
	return &BeaconBlockExecutionRequestConsolidation{
		log:   log.WithField("event", BeaconBlockExecutionRequestConsolidationType),
		event: event,
	}
}

func (b *BeaconBlockExecutionRequestConsolidation) Type() string {
	return BeaconBlockExecutionRequestConsolidationType
}

func (b *BeaconBlockExecutionRequestConsolidation) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestConsolidation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockExecutionRequestConsolidation) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockExecutionRequestConsolidation) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
