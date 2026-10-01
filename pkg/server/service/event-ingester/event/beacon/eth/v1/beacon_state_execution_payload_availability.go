package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	BeaconStateExecutionPayloadAvailabilityType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_EXECUTION_PAYLOAD_AVAILABILITY.String()
)

type BeaconStateExecutionPayloadAvailability struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBeaconStateExecutionPayloadAvailability(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BeaconStateExecutionPayloadAvailability {
	return &BeaconStateExecutionPayloadAvailability{
		log:   log.WithField("event", BeaconStateExecutionPayloadAvailabilityType),
		event: event,
	}
}

func (b *BeaconStateExecutionPayloadAvailability) Type() string {
	return BeaconStateExecutionPayloadAvailabilityType
}

func (b *BeaconStateExecutionPayloadAvailability) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStateExecutionPayloadAvailability)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconStateExecutionPayloadAvailability) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconStateExecutionPayloadAvailability) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
