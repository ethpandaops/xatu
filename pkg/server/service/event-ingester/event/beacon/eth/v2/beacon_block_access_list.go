package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconBlockAccessListType = "BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST"
)

type BeaconBlockAccessList struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockAccessList(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconBlockAccessList {
	return &BeaconBlockAccessList{
		log:   log.WithField("event", BeaconBlockAccessListType),
		event: event,
	}
}

func (b *BeaconBlockAccessList) Type() string {
	return BeaconBlockAccessListType
}

func (b *BeaconBlockAccessList) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockAccessList)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockAccessList) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockAccessList) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
