package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockExecutionRequestDepositEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_DEPOSIT,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockExecutionRequestDepositTableName,
		canonicalBeaconBlockExecutionRequestDepositEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockExecutionRequestDepositBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockExecutionRequestDepositBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockExecutionRequestDeposit() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_execution_request_deposit payload: %w", route.ErrInvalidEvent)
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

func (b *canonicalBeaconBlockExecutionRequestDepositBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV2BeaconBlockExecutionRequestDeposit()

	if payload.GetPubkey() == nil {
		return fmt.Errorf("nil Pubkey: %w", route.ErrInvalidEvent)
	}

	if payload.GetWithdrawalCredentials() == nil {
		return fmt.Errorf("nil WithdrawalCredentials: %w", route.ErrInvalidEvent)
	}

	if payload.GetAmount() == nil {
		return fmt.Errorf("nil Amount: %w", route.ErrInvalidEvent)
	}

	if payload.GetSignature() == nil {
		return fmt.Errorf("nil Signature: %w", route.ErrInvalidEvent)
	}

	if payload.GetIndex() == nil {
		return fmt.Errorf("nil Index: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconBlockExecutionRequestDepositBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockExecutionRequestDepositBatch) appendPayload(event *xatu.DecoratedEvent) {
	deposit := event.GetEthV2BeaconBlockExecutionRequestDeposit()

	b.Pubkey.Append(deposit.GetPubkey().GetValue())
	b.WithdrawalCredentials.Append([]byte(deposit.GetWithdrawalCredentials().GetValue()))
	b.Amount.Append(proto.UInt128{Low: deposit.GetAmount().GetValue()})
	b.Signature.Append(deposit.GetSignature().GetValue())
	b.Index.Append(deposit.GetIndex().GetValue())
}

//nolint:gosec // G115: proto uint64 position is bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockExecutionRequestDepositBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestDeposit()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockVersion.Append("")
		b.BlockRoot.Append(nil)
		b.PositionInBlock.Append(0)

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)

	b.PositionInBlock.Append(uint32(additional.GetPositionInBlock().GetValue()))
}
