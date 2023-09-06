package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockBLSToExecutionChangeType = "BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE"
)

type BeaconBlockBLSToExecutionChange struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockBLSToExecutionChange(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockBLSToExecutionChange {
	return &BeaconBlockBLSToExecutionChange{
		log:   log.WithField("event", BeaconBlockBLSToExecutionChangeType),
		event: event,
	}
}

func (b *BeaconBlockBLSToExecutionChange) Type() string {
	return BeaconBlockBLSToExecutionChangeType
}

func (b *BeaconBlockBLSToExecutionChange) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockBlsToExecutionChange)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockBLSToExecutionChange) Filter(ctx context.Context) bool {
	return false
}
