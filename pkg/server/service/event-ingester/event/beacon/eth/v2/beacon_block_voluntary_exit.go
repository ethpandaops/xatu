package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconBlockVoluntaryExitType = "BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT"
)

type BeaconBlockVoluntaryExit struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockVoluntaryExit(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconBlockVoluntaryExit {
	return &BeaconBlockVoluntaryExit{
		log:   log.WithField("event", BeaconBlockVoluntaryExitType),
		event: event,
	}
}

func (b *BeaconBlockVoluntaryExit) Type() string {
	return BeaconBlockVoluntaryExitType
}

func (b *BeaconBlockVoluntaryExit) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockVoluntaryExit) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockVoluntaryExit) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
