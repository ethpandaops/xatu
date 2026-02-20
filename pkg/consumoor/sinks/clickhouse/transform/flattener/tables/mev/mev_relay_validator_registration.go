package mev

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var mevRelayValidatorRegistrationEventNames = []xatu.Event_Name{
	xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		mevRelayValidatorRegistrationTableName,
		mevRelayValidatorRegistrationEventNames,
		func() flattener.ColumnarBatch { return newmevRelayValidatorRegistrationBatch() },
	))
}

func (b *mevRelayValidatorRegistrationBatch) FlattenTo(
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

func (b *mevRelayValidatorRegistrationBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *mevRelayValidatorRegistrationBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetMevRelayValidatorRegistration()
	if payload == nil {
		b.appendZeroPayload()

		return
	}

	msg := payload.GetMessage()
	if msg == nil {
		b.appendZeroPayload()

		return
	}

	if ts := msg.GetTimestamp(); ts != nil {
		b.Timestamp.Append(int64(ts.GetValue())) //nolint:gosec // proto uint64 narrowed to int64 target field
	} else {
		b.Timestamp.Append(0)
	}

	if gasLimit := msg.GetGasLimit(); gasLimit != nil {
		b.GasLimit.Append(gasLimit.GetValue())
	} else {
		b.GasLimit.Append(0)
	}

	if feeRecipient := msg.GetFeeRecipient(); feeRecipient != nil {
		b.FeeRecipient.Append(feeRecipient.GetValue())
	} else {
		b.FeeRecipient.Append("")
	}
}

func (b *mevRelayValidatorRegistrationBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.appendZeroAdditionalData()

		return
	}

	client := event.GetMeta().GetClient()
	additional := client.GetMevRelayValidatorRegistration()

	if additional == nil {
		b.appendZeroAdditionalData()

		return
	}

	if relay := additional.GetRelay(); relay != nil {
		if name := relay.GetName(); name != nil {
			b.RelayName.Append(name.GetValue())
		} else {
			b.RelayName.Append("")
		}
	} else {
		b.RelayName.Append("")
	}

	if validatorIndex := additional.GetValidatorIndex(); validatorIndex != nil {
		b.ValidatorIndex.Append(uint32(validatorIndex.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
	} else {
		b.ValidatorIndex.Append(0)
	}

	if slot := additional.GetSlot(); slot != nil {
		if slotNumber := slot.GetNumber(); slotNumber != nil {
			b.Slot.Append(uint32(slotNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
		} else {
			b.Slot.Append(0)
		}

		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
	}

	if epoch := additional.GetEpoch(); epoch != nil {
		if epochNumber := epoch.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
		} else {
			b.Epoch.Append(0)
		}

		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
	}

	if wallclockSlot := additional.GetWallclockSlot(); wallclockSlot != nil {
		if slotNumber := wallclockSlot.GetNumber(); slotNumber != nil {
			b.WallclockSlot.Append(uint32(slotNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
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

	if wallclockEpoch := additional.GetWallclockEpoch(); wallclockEpoch != nil {
		if epochNumber := wallclockEpoch.GetNumber(); epochNumber != nil {
			b.WallclockEpoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
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

func (b *mevRelayValidatorRegistrationBatch) appendZeroPayload() {
	b.Timestamp.Append(0)
	b.GasLimit.Append(0)
	b.FeeRecipient.Append("")
}

func (b *mevRelayValidatorRegistrationBatch) appendZeroAdditionalData() {
	b.RelayName.Append("")
	b.ValidatorIndex.Append(0)
	b.Slot.Append(0)
	b.SlotStartDateTime.Append(time.Time{})
	b.Epoch.Append(0)
	b.EpochStartDateTime.Append(time.Time{})
	b.WallclockSlot.Append(0)
	b.WallclockSlotStartDateTime.Append(time.Time{})
	b.WallclockEpoch.Append(0)
	b.WallclockEpochStartDateTime.Append(time.Time{})
}
