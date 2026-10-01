package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconBlockExecutionRequestBuilderExitType = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_EXIT"
)

type BeaconBlockExecutionRequestBuilderExit struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockExecutionRequestBuilderExit(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconBlockExecutionRequestBuilderExit {
	return &BeaconBlockExecutionRequestBuilderExit{
		log:   log.WithField("event", BeaconBlockExecutionRequestBuilderExitType),
		event: event,
	}
}

func (b *BeaconBlockExecutionRequestBuilderExit) Type() string {
	return BeaconBlockExecutionRequestBuilderExitType
}

func (b *BeaconBlockExecutionRequestBuilderExit) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestBuilderExit)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockExecutionRequestBuilderExit) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockExecutionRequestBuilderExit) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
