package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconBlockExecutionRequestWithdrawalType = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_WITHDRAWAL"
)

type BeaconBlockExecutionRequestWithdrawal struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockExecutionRequestWithdrawal(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconBlockExecutionRequestWithdrawal {
	return &BeaconBlockExecutionRequestWithdrawal{
		log:   log.WithField("event", BeaconBlockExecutionRequestWithdrawalType),
		event: event,
	}
}

func (b *BeaconBlockExecutionRequestWithdrawal) Type() string {
	return BeaconBlockExecutionRequestWithdrawalType
}

func (b *BeaconBlockExecutionRequestWithdrawal) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestWithdrawal)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockExecutionRequestWithdrawal) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockExecutionRequestWithdrawal) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
