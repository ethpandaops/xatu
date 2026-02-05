package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockSyncAggregateType = "BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE"
)

type BeaconBlockSyncAggregate struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockSyncAggregate(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockSyncAggregate {
	return &BeaconBlockSyncAggregate{
		log:   log.WithField("event", BeaconBlockSyncAggregateType),
		event: event,
	}
}

func (b *BeaconBlockSyncAggregate) Type() string {
	return BeaconBlockSyncAggregateType
}

func (b *BeaconBlockSyncAggregate) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockSyncAggregate)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockSyncAggregate) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconBlockSyncAggregate) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
