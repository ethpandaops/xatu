package mev

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		mevRelayValidatorRegistrationTableName,
		[]xatu.Event_Name{xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION},
		func() flattener.ColumnarBatch {
			return newmevRelayValidatorRegistrationBatch()
		},
	))
}

func (b *mevRelayValidatorRegistrationBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetMevRelayValidatorRegistration()
	if payload == nil {
		return fmt.Errorf("nil MevRelayValidatorRegistration payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetMevRelayValidatorRegistration()
	msg := payload.GetMessage()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Timestamp.Append(int64(msg.GetTimestamp().GetValue())) //nolint:gosec // G115: timestamp fits int64.
	b.RelayName.Append(addl.GetRelay().GetName().GetValue())
	b.ValidatorIndex.Append(uint32(addl.GetValidatorIndex().GetValue())) //nolint:gosec // G115: validator index fits uint32.
	b.GasLimit.Append(msg.GetGasLimit().GetValue())
	b.FeeRecipient.Append(msg.GetFeeRecipient().GetValue())
	b.Slot.Append(uint32(addl.GetSlot().GetNumber().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.WallclockSlot.Append(uint32(addl.GetWallclockSlot().GetNumber().GetValue())) //nolint:gosec // G115: wallclock slot fits uint32.
	b.WallclockSlotStartDateTime.Append(addl.GetWallclockSlot().GetStartDateTime().AsTime())
	b.WallclockEpoch.Append(uint32(addl.GetWallclockEpoch().GetNumber().GetValue())) //nolint:gosec // G115: wallclock epoch fits uint32.
	b.WallclockEpochStartDateTime.Append(addl.GetWallclockEpoch().GetStartDateTime().AsTime())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *mevRelayValidatorRegistrationBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetMevRelayValidatorRegistration()

	if payload.GetMessage() == nil {
		return fmt.Errorf("nil Message: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetMevRelayValidatorRegistration()
	if addl == nil {
		return fmt.Errorf("nil MevRelayValidatorRegistration additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil || addl.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetValidatorIndex() == nil {
		return fmt.Errorf("nil ValidatorIndex: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
