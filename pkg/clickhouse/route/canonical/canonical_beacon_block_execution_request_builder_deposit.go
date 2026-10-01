package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockExecutionRequestBuilderDepositEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockExecutionRequestBuilderDepositTableName,
		canonicalBeaconBlockExecutionRequestBuilderDepositEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockExecutionRequestBuilderDepositBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockExecutionRequestBuilderDepositBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockExecutionRequestBuilderDeposit() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_execution_request_builder_deposit payload: %w", route.ErrInvalidEvent)
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

func (b *canonicalBeaconBlockExecutionRequestBuilderDepositBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV2BeaconBlockExecutionRequestBuilderDeposit()

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

	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderDeposit()
	if additional == nil || additional.GetPositionInBlock() == nil {
		return fmt.Errorf("nil PositionInBlock: %w", route.ErrInvalidEvent)
	}

	if additional.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", route.ErrInvalidEvent)
	}

	if additional.GetBlockHash() == "" {
		return fmt.Errorf("empty BlockHash: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconBlockExecutionRequestBuilderDepositBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockExecutionRequestBuilderDepositBatch) appendPayload(event *xatu.DecoratedEvent) {
	deposit := event.GetEthV2BeaconBlockExecutionRequestBuilderDeposit()

	b.Pubkey.Append(deposit.GetPubkey().GetValue())
	b.WithdrawalCredentials.Append([]byte(deposit.GetWithdrawalCredentials().GetValue()))
	b.Amount.Append(proto.UInt128{Low: deposit.GetAmount().GetValue()})
	b.Signature.Append(deposit.GetSignature().GetValue())
}

//nolint:gosec // G115: proto uint64 position is bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockExecutionRequestBuilderDepositBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderDeposit()
	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)

	b.BlockNumber.Append(additional.GetBlockNumber().GetValue())
	b.BlockHash.Append([]byte(additional.GetBlockHash()))
	b.PositionInBlock.Append(uint32(additional.GetPositionInBlock().GetValue()))
}
