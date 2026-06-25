package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconBlockExecutionRequestDepositType = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_DEPOSIT"
)

type BeaconBlockExecutionRequestDeposit struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockExecutionRequestDeposit(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconBlockExecutionRequestDeposit {
	return &BeaconBlockExecutionRequestDeposit{
		log:   log.WithField("event", BeaconBlockExecutionRequestDepositType),
		event: event,
	}
}

func (b *BeaconBlockExecutionRequestDeposit) Type() string {
	return BeaconBlockExecutionRequestDepositType
}

func (b *BeaconBlockExecutionRequestDeposit) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestDeposit)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockExecutionRequestDeposit) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockExecutionRequestDeposit) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
