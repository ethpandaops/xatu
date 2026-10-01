package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStateBuilderPendingPaymentEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_PAYMENT,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStateBuilderPendingPaymentTableName,
		canonicalBeaconStateBuilderPendingPaymentEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStateBuilderPendingPaymentBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStateBuilderPendingPaymentBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStateBuilderPendingPayment()
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

func (b *canonicalBeaconStateBuilderPendingPaymentBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconStateBuilderPendingPayment()

	if payload.GetPaymentIndex() == nil {
		return fmt.Errorf("nil PaymentIndex: %w", route.ErrInvalidEvent)
	}

	if payload.GetWeight() == nil {
		return fmt.Errorf("nil Weight: %w", route.ErrInvalidEvent)
	}

	if payload.GetFeeRecipient() == "" {
		return fmt.Errorf("nil FeeRecipient: %w", route.ErrInvalidEvent)
	}

	if payload.GetAmount() == nil {
		return fmt.Errorf("nil Amount: %w", route.ErrInvalidEvent)
	}

	if payload.GetBuilderIndex() == nil {
		return fmt.Errorf("nil BuilderIndex: %w", route.ErrInvalidEvent)
	}

	if payload.GetProposerIndex() == nil {
		return fmt.Errorf("nil ProposerIndex: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStateBuilderPendingPayment()
	if extra == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	if err := validateSlotData(extra.GetSlot()); err != nil {
		return err
	}

	return validateEpochData(extra.GetEpoch())
}

func (b *canonicalBeaconStateBuilderPendingPaymentBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStateBuilderPendingPaymentBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconStateBuilderPendingPayment()

	b.PaymentIndex.Append(uint32(payload.GetPaymentIndex().GetValue())) //nolint:gosec // bounded by uint32 column
	b.Weight.Append(payload.GetWeight().GetValue())
	b.FeeRecipient.Append([]byte(payload.GetFeeRecipient()))
	b.Amount.Append(payload.GetAmount().GetValue())
	b.BuilderIndex.Append(payload.GetBuilderIndex().GetValue())
	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue())) //nolint:gosec // bounded by uint32 column
}

func (b *canonicalBeaconStateBuilderPendingPaymentBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStateBuilderPendingPayment()

	b.Epoch.Append(uint32(extra.GetEpoch().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.EpochStartDateTime.Append(timeOrZero(extra.GetEpoch().GetStartDateTime()))
	b.Slot.Append(uint32(extra.GetSlot().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.SlotStartDateTime.Append(timeOrZero(extra.GetSlot().GetStartDateTime()))
	b.StateID.Append(extra.GetStateId())
}
