package mev

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var mevRelayValidatorRegistrationEventNames = []xatu.Event_Name{
	xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION,
}

func init() {
	r, err := route.NewStaticRoute(
		mevRelayValidatorRegistrationTableName,
		mevRelayValidatorRegistrationEventNames,
		func() route.ColumnarBatch { return newmevRelayValidatorRegistrationBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *mevRelayValidatorRegistrationBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetMevRelayValidatorRegistration() == nil {
		return fmt.Errorf("nil mev_relay_validator_registration payload: %w", route.ErrInvalidEvent)
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

func (b *mevRelayValidatorRegistrationBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetMevRelayValidatorRegistration()

	if msg := payload.GetMessage(); msg != nil {
		if msg.GetTimestamp() == nil {
			return fmt.Errorf("nil Timestamp: %w", route.ErrInvalidEvent)
		}

		if msg.GetGasLimit() == nil {
			return fmt.Errorf("nil GasLimit: %w", route.ErrInvalidEvent)
		}
	}

	if client := event.GetMeta().GetClient(); client != nil {
		if additional := client.GetMevRelayValidatorRegistration(); additional != nil {
			if additional.GetValidatorIndex() == nil {
				return fmt.Errorf("nil ValidatorIndex: %w", route.ErrInvalidEvent)
			}

			if slot := additional.GetSlot(); slot != nil {
				if slot.GetNumber() == nil {
					return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
				}
			}

			if epoch := additional.GetEpoch(); epoch != nil {
				if epoch.GetNumber() == nil {
					return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
				}
			}

			if wallclockSlot := additional.GetWallclockSlot(); wallclockSlot != nil {
				if wallclockSlot.GetNumber() == nil {
					return fmt.Errorf("nil WallclockSlot: %w", route.ErrInvalidEvent)
				}
			}

			if wallclockEpoch := additional.GetWallclockEpoch(); wallclockEpoch != nil {
				if wallclockEpoch.GetNumber() == nil {
					return fmt.Errorf("nil WallclockEpoch: %w", route.ErrInvalidEvent)
				}
			}
		}
	}

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
