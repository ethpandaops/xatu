package canonical

import (
	"fmt"
	"time"

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

	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingPartialWithdrawal()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	epoch := uint32(extra.GetEpoch().GetNumber().GetValue()) //nolint:gosec // bounded by uint32 column

	var epochStartTime time.Time
	if start := extra.GetEpoch().GetStartDateTime(); start != nil {
		epochStartTime = start.AsTime()
	}

	b.UpdatedDateTime.Append(time.Now())
	b.Epoch.Append(epoch)
	b.EpochStartDateTime.Append(epochStartTime)
	b.StateID.Append(extra.GetStateId())
	b.PositionInQueue.Append(uint32(extra.GetPositionInQueue().GetValue())) //nolint:gosec // bounded by uint32 column
	b.ValidatorIndex.Append(uint32(payload.GetValidatorIndex().GetValue())) //nolint:gosec // bounded by uint32 column
	b.Amount.Append(payload.GetAmount().GetValue())
	b.WithdrawableEpoch.Append(payload.GetWithdrawableEpoch().GetValue())
	b.appendMetadata(event)
	b.rows++

	return nil
}
