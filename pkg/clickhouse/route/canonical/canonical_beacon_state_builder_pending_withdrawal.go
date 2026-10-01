package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStateBuilderPendingWithdrawalEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_WITHDRAWAL,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStateBuilderPendingWithdrawalTableName,
		canonicalBeaconStateBuilderPendingWithdrawalEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStateBuilderPendingWithdrawalBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStateBuilderPendingWithdrawalBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStateBuilderPendingWithdrawal()
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

func (b *canonicalBeaconStateBuilderPendingWithdrawalBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconStateBuilderPendingWithdrawal()

	if payload.GetFeeRecipient() == "" {
		return fmt.Errorf("nil FeeRecipient: %w", route.ErrInvalidEvent)
	}

	if payload.GetAmount() == nil {
		return fmt.Errorf("nil Amount: %w", route.ErrInvalidEvent)
	}

	if payload.GetBuilderIndex() == nil {
		return fmt.Errorf("nil BuilderIndex: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStateBuilderPendingWithdrawal()
	if extra == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	if err := validateEpochData(extra.GetEpoch()); err != nil {
		return err
	}

	if extra.GetPositionInQueue() == nil {
		return fmt.Errorf("nil PositionInQueue: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconStateBuilderPendingWithdrawalBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStateBuilderPendingWithdrawalBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconStateBuilderPendingWithdrawal()

	b.FeeRecipient.Append([]byte(payload.GetFeeRecipient()))
	b.Amount.Append(payload.GetAmount().GetValue())
	b.BuilderIndex.Append(payload.GetBuilderIndex().GetValue())
}

func (b *canonicalBeaconStateBuilderPendingWithdrawalBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStateBuilderPendingWithdrawal()

	b.Epoch.Append(uint32(extra.GetEpoch().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.EpochStartDateTime.Append(timeOrZero(extra.GetEpoch().GetStartDateTime()))
	b.StateID.Append(extra.GetStateId())
	b.PositionInQueue.Append(uint32(extra.GetPositionInQueue().GetValue())) //nolint:gosec // bounded by uint32 column
}
