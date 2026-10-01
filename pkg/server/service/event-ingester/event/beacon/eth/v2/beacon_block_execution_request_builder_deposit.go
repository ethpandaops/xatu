package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconBlockExecutionRequestBuilderDepositType = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT"
)

type BeaconBlockExecutionRequestBuilderDeposit struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockExecutionRequestBuilderDeposit(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconBlockExecutionRequestBuilderDeposit {
	return &BeaconBlockExecutionRequestBuilderDeposit{
		log:   log.WithField("event", BeaconBlockExecutionRequestBuilderDepositType),
		event: event,
	}
}

func (b *BeaconBlockExecutionRequestBuilderDeposit) Type() string {
	return BeaconBlockExecutionRequestBuilderDepositType
}

func (b *BeaconBlockExecutionRequestBuilderDeposit) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestBuilderDeposit)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockExecutionRequestBuilderDeposit) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockExecutionRequestBuilderDeposit) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
