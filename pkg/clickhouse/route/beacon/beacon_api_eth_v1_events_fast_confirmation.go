package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

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
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.WallclockSlot.Append(0)
		b.WallclockSlotStartDateTime.Append(time.Time{})
		b.WallclockEpoch.Append(0)
		b.WallclockEpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additionalV2 := client.GetEthV1EventsFastConfirmation()
	additional := extractBeaconSlotEpochPropagation(additionalV2)

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))

	if wallclockSlot := additionalV2.GetWallclockSlot(); wallclockSlot != nil {
		if n := wallclockSlot.GetNumber(); n != nil {
			b.WallclockSlot.Append(uint32(n.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.WallclockSlot.Append(0)
		}

		if startDateTime := wallclockSlot.GetStartDateTime(); startDateTime != nil {
			b.WallclockSlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.WallclockSlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.WallclockSlot.Append(0)
		b.WallclockSlotStartDateTime.Append(time.Time{})
	}

	if wallclockEpoch := additionalV2.GetWallclockEpoch(); wallclockEpoch != nil {
		if n := wallclockEpoch.GetNumber(); n != nil {
			b.WallclockEpoch.Append(uint32(n.GetValue())) //nolint:gosec // epoch fits uint32
		} else {
			b.WallclockEpoch.Append(0)
		}

		if startDateTime := wallclockEpoch.GetStartDateTime(); startDateTime != nil {
			b.WallclockEpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.WallclockEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.WallclockEpoch.Append(0)
		b.WallclockEpochStartDateTime.Append(time.Time{})
	}
}
