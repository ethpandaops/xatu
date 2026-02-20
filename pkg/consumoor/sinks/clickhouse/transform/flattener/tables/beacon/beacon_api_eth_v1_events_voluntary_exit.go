package beacon

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsVoluntaryExitEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsVoluntaryExitTableName,
		beaconApiEthV1EventsVoluntaryExitEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1EventsVoluntaryExitBatch() },
	))
}

func (b *beaconApiEthV1EventsVoluntaryExitBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsVoluntaryExitBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsVoluntaryExitBatch) appendPayload(event *xatu.DecoratedEvent) {
	voluntaryExitV2 := event.GetEthV1EventsVoluntaryExitV2()
	if voluntaryExitV2 == nil {
		b.Signature.Append("")
		b.ValidatorIndex.Append(0)
		b.Epoch.Append(0)

		return
	}

	b.Signature.Append(voluntaryExitV2.GetSignature())

	message := voluntaryExitV2.GetMessage()
	if message == nil {
		b.ValidatorIndex.Append(0)
		b.Epoch.Append(0)

		return
	}

	if validatorIndex := message.GetValidatorIndex(); validatorIndex != nil {
		b.ValidatorIndex.Append(uint32(validatorIndex.GetValue())) //nolint:gosec // validator index fits uint32
	} else {
		b.ValidatorIndex.Append(0)
	}

	if epoch := message.GetEpoch(); epoch != nil {
		b.Epoch.Append(uint32(epoch.GetValue())) //nolint:gosec // epoch fits uint32
	} else {
		b.Epoch.Append(0)
	}
}

func (b *beaconApiEthV1EventsVoluntaryExitBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.EpochStartDateTime.Append(time.Time{})
		b.WallclockSlot.Append(0)
		b.WallclockSlotStartDateTime.Append(time.Time{})
		b.WallclockEpoch.Append(0)
		b.WallclockEpochStartDateTime.Append(time.Time{})

		return
	}

	additionalV2 := event.GetMeta().GetClient().GetEthV1EventsVoluntaryExitV2()
	if additionalV2 == nil {
		b.EpochStartDateTime.Append(time.Time{})
		b.WallclockSlot.Append(0)
		b.WallclockSlotStartDateTime.Append(time.Time{})
		b.WallclockEpoch.Append(0)
		b.WallclockEpochStartDateTime.Append(time.Time{})

		return
	}

	if epoch := additionalV2.GetEpoch(); epoch != nil {
		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.EpochStartDateTime.Append(time.Time{})
	}

	if wallclockSlotNumber := additionalV2.GetWallclockSlot().GetNumber(); wallclockSlotNumber != nil {
		b.WallclockSlot.Append(uint32(wallclockSlotNumber.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.WallclockSlot.Append(0)
	}

	if wallclockSlot := additionalV2.GetWallclockSlot(); wallclockSlot != nil {
		if startDateTime := wallclockSlot.GetStartDateTime(); startDateTime != nil {
			b.WallclockSlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.WallclockSlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.WallclockSlotStartDateTime.Append(time.Time{})
	}

	// WallclockEpoch mirrors the epoch value already appended to b.Epoch.
	if epochNumber := additionalV2.GetEpoch().GetNumber(); epochNumber != nil {
		b.WallclockEpoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // epoch fits uint32
	} else {
		b.WallclockEpoch.Append(0)
	}

	if wallclockEpoch := additionalV2.GetWallclockEpoch(); wallclockEpoch != nil {
		if startDateTime := wallclockEpoch.GetStartDateTime(); startDateTime != nil {
			b.WallclockEpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.WallclockEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.WallclockEpochStartDateTime.Append(time.Time{})
	}
}
