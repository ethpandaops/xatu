package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconBlockAccessListSummaryType = "BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST_SUMMARY"
)

type BeaconBlockAccessListSummary struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockAccessListSummary(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconBlockAccessListSummary {
	return &BeaconBlockAccessListSummary{
		log:   log.WithField("event", BeaconBlockAccessListSummaryType),
		event: event,
	}
}

func (b *BeaconBlockAccessListSummary) Type() string {
	return BeaconBlockAccessListSummaryType
}

func (b *BeaconBlockAccessListSummary) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockAccessListSummary)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockAccessListSummary) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockAccessListSummary) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
