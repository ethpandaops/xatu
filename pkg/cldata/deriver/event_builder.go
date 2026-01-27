package deriver

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// EventBuilder provides helper methods for constructing decorated events.
type EventBuilder struct {
	ctxProvider cldata.ContextProvider
}

// NewEventBuilder creates a new event builder.
func NewEventBuilder(ctxProvider cldata.ContextProvider) *EventBuilder {
	return &EventBuilder{ctxProvider: ctxProvider}
}

// CreateDecoratedEvent creates a new decorated event with common fields populated.
func (b *EventBuilder) CreateDecoratedEvent(
	ctx context.Context,
	eventName xatu.Event_Name,
) (*xatu.DecoratedEvent, error) {
	clientMeta, err := b.ctxProvider.CreateClientMeta(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client metadata")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     eventName,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
	}, nil
}

// BuildSlotV2 creates a SlotV2 from a slot number using the wallclock.
func (b *EventBuilder) BuildSlotV2(slotNum uint64) *xatu.SlotV2 {
	slot := b.ctxProvider.Wallclock().Slots().FromNumber(slotNum)

	return &xatu.SlotV2{
		Number:        &wrapperspb.UInt64Value{Value: slotNum},
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
	}
}

// BuildEpochV2 creates an EpochV2 from an epoch number using the wallclock.
func (b *EventBuilder) BuildEpochV2(epochNum uint64) *xatu.EpochV2 {
	epoch := b.ctxProvider.Wallclock().Epochs().FromNumber(epochNum)

	return &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epochNum},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}
}

// BuildEpochV2FromSlot creates an EpochV2 from a slot number using the wallclock.
func (b *EventBuilder) BuildEpochV2FromSlot(slotNum uint64) *xatu.EpochV2 {
	epoch := b.ctxProvider.Wallclock().Epochs().FromSlot(slotNum)

	return &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}
}
