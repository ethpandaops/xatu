package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStateExecutionPayloadAvailabilityEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_EXECUTION_PAYLOAD_AVAILABILITY,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStateExecutionPayloadAvailabilityTableName,
		canonicalBeaconStateExecutionPayloadAvailabilityEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStateExecutionPayloadAvailabilityBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStateExecutionPayloadAvailabilityBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStateExecutionPayloadAvailability()
	if payload == nil {
		return fmt.Errorf("nil payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconStateExecutionPayloadAvailabilityBatch) validate(event *xatu.DecoratedEvent) error {
	if event.GetEthV1BeaconStateExecutionPayloadAvailability().GetAvailable() == nil {
		return fmt.Errorf("nil Available: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStateExecutionPayloadAvailability()
	if extra == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	if err := validateSlotData(extra.GetSlot()); err != nil {
		return err
	}

	return validateEpochData(extra.GetEpoch())
}

func (b *canonicalBeaconStateExecutionPayloadAvailabilityBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStateExecutionPayloadAvailabilityBatch) appendPayload(event *xatu.DecoratedEvent) {
	b.Available.Append(event.GetEthV1BeaconStateExecutionPayloadAvailability().GetAvailable().GetValue())
}

func (b *canonicalBeaconStateExecutionPayloadAvailabilityBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStateExecutionPayloadAvailability()

	b.Slot.Append(uint32(extra.GetSlot().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.SlotStartDateTime.Append(timeOrZero(extra.GetSlot().GetStartDateTime()))
	b.Epoch.Append(uint32(extra.GetEpoch().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.EpochStartDateTime.Append(timeOrZero(extra.GetEpoch().GetStartDateTime()))
	b.StateID.Append(extra.GetStateId())
}
