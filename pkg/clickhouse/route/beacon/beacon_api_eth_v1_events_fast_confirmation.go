package beacon

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func appendNullableNumberAndStart(
	numCol *proto.ColNullable[uint32],
	timeCol *proto.ColNullable[time.Time],
	number *wrapperspb.UInt64Value,
	start *timestamppb.Timestamp,
) {
	if number != nil {
		numCol.Append(proto.NewNullable(uint32(number.GetValue()))) //nolint:gosec // slot/epoch fits uint32
	} else {
		numCol.Append(proto.Nullable[uint32]{})
	}

	if start != nil {
		timeCol.Append(proto.NewNullable(start.AsTime()))
	} else {
		timeCol.Append(proto.Nullable[time.Time]{})
	}
}

var beaconApiEthV1EventsFastConfirmationEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_FAST_CONFIRMATION,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsFastConfirmationTableName,
		beaconApiEthV1EventsFastConfirmationEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsFastConfirmationBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsFastConfirmationBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsFastConfirmation() == nil {
		return fmt.Errorf("nil eth_v1_events_fast_confirmation payload: %w", route.ErrInvalidEvent)
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

func (b *beaconApiEthV1EventsFastConfirmationBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsFastConfirmation()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *beaconApiEthV1EventsFastConfirmationBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsFastConfirmationBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1EventsFastConfirmation()
	if slot := payload.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	b.Block.Append([]byte(payload.GetBlock()))
}

func (b *beaconApiEthV1EventsFastConfirmationBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(proto.Nullable[uint32]{})
		b.Epoch.Append(proto.Nullable[uint32]{})
		b.EpochStartDateTime.Append(proto.Nullable[time.Time]{})
		b.WallclockSlot.Append(proto.Nullable[uint32]{})
		b.WallclockSlotStartDateTime.Append(proto.Nullable[time.Time]{})
		b.WallclockEpoch.Append(proto.Nullable[uint32]{})
		b.WallclockEpochStartDateTime.Append(proto.Nullable[time.Time]{})

		return
	}

	additionalV2 := event.GetMeta().GetClient().GetEthV1EventsFastConfirmation()

	if slot := additionalV2.GetSlot(); slot != nil {
		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.SlotStartDateTime.Append(time.Time{})
	}

	if propagation := additionalV2.GetPropagation(); propagation != nil {
		if diff := propagation.GetSlotStartDiff(); diff != nil {
			b.PropagationSlotStartDiff.Append(proto.NewNullable(uint32(diff.GetValue()))) //nolint:gosec // diff fits uint32
		} else {
			b.PropagationSlotStartDiff.Append(proto.Nullable[uint32]{})
		}
	} else {
		b.PropagationSlotStartDiff.Append(proto.Nullable[uint32]{})
	}

	appendNullableNumberAndStart(b.Epoch, b.EpochStartDateTime, additionalV2.GetEpoch().GetNumber(), additionalV2.GetEpoch().GetStartDateTime())
	appendNullableNumberAndStart(b.WallclockSlot, b.WallclockSlotStartDateTime, additionalV2.GetWallclockSlot().GetNumber(), additionalV2.GetWallclockSlot().GetStartDateTime())
	appendNullableNumberAndStart(b.WallclockEpoch, b.WallclockEpochStartDateTime, additionalV2.GetWallclockEpoch().GetNumber(), additionalV2.GetWallclockEpoch().GetStartDateTime())
}
