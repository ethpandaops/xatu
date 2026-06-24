package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStatePendingDepositEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStatePendingDepositTableName,
		canonicalBeaconStatePendingDepositEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStatePendingDepositBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStatePendingDepositBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStatePendingDeposit()
	if payload == nil {
		return fmt.Errorf("nil payload: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingDeposit()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	var epochStartTime time.Time
	if start := extra.GetEpoch().GetStartDateTime(); start != nil {
		epochStartTime = start.AsTime()
	}

	b.UpdatedDateTime.Append(time.Now())
	b.Epoch.Append(uint32(extra.GetEpoch().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.EpochStartDateTime.Append(epochStartTime)
	b.StateID.Append(extra.GetStateId())
	b.PositionInQueue.Append(uint32(extra.GetPositionInQueue().GetValue())) //nolint:gosec // bounded by uint32 column
	b.Pubkey.Append(payload.GetPubkey())
	b.WithdrawalCredentials.Append([]byte(payload.GetWithdrawalCredentials()))
	b.Amount.Append(proto.UInt128{Low: payload.GetAmount().GetValue()})
	b.Signature.Append(payload.GetSignature())
	b.Slot.Append(uint32(payload.GetSlot().GetValue()))

	b.appendMetadata(event)
	b.rows++

	return nil
}
