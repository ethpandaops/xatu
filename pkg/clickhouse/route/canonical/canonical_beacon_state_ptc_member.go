package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStatePtcMemberEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PTC_MEMBER,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStatePtcMemberTableName,
		canonicalBeaconStatePtcMemberEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStatePtcMemberBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStatePtcMemberBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStatePtcMember()
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

func (b *canonicalBeaconStatePtcMemberBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconStatePtcMember()

	if payload.GetPosition() == nil {
		return fmt.Errorf("nil Position: %w", route.ErrInvalidEvent)
	}

	if payload.GetValidatorIndex() == nil {
		return fmt.Errorf("nil ValidatorIndex: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePtcMember()
	if extra == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	if err := validateSlotData(extra.GetSlot()); err != nil {
		return err
	}

	return validateEpochData(extra.GetEpoch())
}

func (b *canonicalBeaconStatePtcMemberBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStatePtcMemberBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconStatePtcMember()

	b.Position.Append(uint32(payload.GetPosition().GetValue()))             //nolint:gosec // bounded by uint32 column
	b.ValidatorIndex.Append(uint32(payload.GetValidatorIndex().GetValue())) //nolint:gosec // bounded by uint32 column
}

func (b *canonicalBeaconStatePtcMemberBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePtcMember()

	b.Slot.Append(uint32(extra.GetSlot().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.SlotStartDateTime.Append(timeOrZero(extra.GetSlot().GetStartDateTime()))
	b.Epoch.Append(uint32(extra.GetEpoch().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.EpochStartDateTime.Append(timeOrZero(extra.GetEpoch().GetStartDateTime()))
	b.StateID.Append(extra.GetStateId())
}
