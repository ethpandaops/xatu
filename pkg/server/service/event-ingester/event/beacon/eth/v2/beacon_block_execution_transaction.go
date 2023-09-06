package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockExecutionTransactionType = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION"
)

type BeaconBlockExecutionTransaction struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockExecutionTransaction(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockExecutionTransaction {
	return &BeaconBlockExecutionTransaction{
		log:   log.WithField("event", BeaconBlockExecutionTransactionType),
		event: event,
	}
}

func (b *BeaconBlockExecutionTransaction) Type() string {
	return BeaconBlockExecutionTransactionType
}

func (b *BeaconBlockExecutionTransaction) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockExecutionTransaction)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockExecutionTransaction) Filter(ctx context.Context) bool {
	return false
}
