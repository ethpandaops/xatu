package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	BeaconStateBuilderType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER.String()
)

type BeaconStateBuilder struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconStateBuilder(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconStateBuilder {
	return &BeaconStateBuilder{
		log:   log.WithField("event", BeaconStateBuilderType),
		event: event,
	}
}

func (b *BeaconStateBuilder) Type() string {
	return BeaconStateBuilderType
}

func (b *BeaconStateBuilder) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStateBuilder)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconStateBuilder) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconStateBuilder) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
