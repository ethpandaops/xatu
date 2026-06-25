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

func (b *canonicalBeaconStatePendingDepositBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconStatePendingDeposit()

	if payload.GetPubkey() == "" {
		return fmt.Errorf("nil Pubkey: %w", route.ErrInvalidEvent)
	}

	if payload.GetWithdrawalCredentials() == "" {
		return fmt.Errorf("nil WithdrawalCredentials: %w", route.ErrInvalidEvent)
	}

	if payload.GetAmount() == nil {
		return fmt.Errorf("nil Amount: %w", route.ErrInvalidEvent)
	}

	if payload.GetSignature() == "" {
		return fmt.Errorf("nil Signature: %w", route.ErrInvalidEvent)
	}

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingDeposit()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	if extra.GetPositionInQueue() == nil {
		return fmt.Errorf("nil PositionInQueue: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconStatePendingDepositBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStatePendingDepositBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconStatePendingDeposit()

	b.Pubkey.Append(payload.GetPubkey())
	b.WithdrawalCredentials.Append([]byte(payload.GetWithdrawalCredentials()))
	b.Amount.Append(proto.UInt128{Low: payload.GetAmount().GetValue()})
	b.Signature.Append(payload.GetSignature())
	//nolint:gosec // G115: proto uint64 slot bounded by ClickHouse uint32 column schema
	b.Slot.Append(uint32(payload.GetSlot().GetValue()))
}

func (b *canonicalBeaconStatePendingDepositBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingDeposit()
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
