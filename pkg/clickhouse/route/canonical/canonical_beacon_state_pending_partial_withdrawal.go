package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStatePendingPartialWithdrawalEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStatePendingPartialWithdrawalTableName,
		canonicalBeaconStatePendingPartialWithdrawalEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStatePendingPartialWithdrawalBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStatePendingPartialWithdrawalBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStatePendingPartialWithdrawal()
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

func (b *canonicalBeaconStatePendingPartialWithdrawalBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconStatePendingPartialWithdrawal()

	if payload.GetValidatorIndex() == nil {
		return fmt.Errorf("nil ValidatorIndex: %w", route.ErrInvalidEvent)
	}

	if payload.GetAmount() == nil {
		return fmt.Errorf("nil Amount: %w", route.ErrInvalidEvent)
	}

	if payload.GetWithdrawableEpoch() == nil {
		return fmt.Errorf("nil WithdrawableEpoch: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingPartialWithdrawal()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	if extra.GetPositionInQueue() == nil {
		return fmt.Errorf("nil PositionInQueue: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconStatePendingPartialWithdrawalBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStatePendingPartialWithdrawalBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconStatePendingPartialWithdrawal()

	b.ValidatorIndex.Append(uint32(payload.GetValidatorIndex().GetValue())) //nolint:gosec // bounded by uint32 column
	b.Amount.Append(proto.UInt128{Low: payload.GetAmount().GetValue()})
	b.WithdrawableEpoch.Append(payload.GetWithdrawableEpoch().GetValue())
}

func (b *canonicalBeaconStatePendingPartialWithdrawalBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingPartialWithdrawal()
	epoch := extra.GetEpoch()

	b.Epoch.Append(uint32(epoch.GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column

	if start := epoch.GetStartDateTime(); start != nil {
		b.EpochStartDateTime.Append(start.AsTime())
	} else {
		b.EpochStartDateTime.Append(time.Time{})
	}

	b.StateID.Append(extra.GetStateId())
	b.PositionInQueue.Append(uint32(extra.GetPositionInQueue().GetValue())) //nolint:gosec // bounded by uint32 column
}
