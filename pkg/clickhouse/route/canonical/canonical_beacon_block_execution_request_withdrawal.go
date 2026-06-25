package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockExecutionRequestWithdrawalEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_WITHDRAWAL,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockExecutionRequestWithdrawalTableName,
		canonicalBeaconBlockExecutionRequestWithdrawalEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockExecutionRequestWithdrawalBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockExecutionRequestWithdrawalBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockExecutionRequestWithdrawal() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_execution_request_withdrawal payload: %w", route.ErrInvalidEvent)
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

func (b *canonicalBeaconBlockExecutionRequestWithdrawalBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV2BeaconBlockExecutionRequestWithdrawal()

	if payload.GetSourceAddress() == nil {
		return fmt.Errorf("nil SourceAddress: %w", route.ErrInvalidEvent)
	}

	if payload.GetValidatorPubkey() == nil {
		return fmt.Errorf("nil ValidatorPubkey: %w", route.ErrInvalidEvent)
	}

	if payload.GetAmount() == nil {
		return fmt.Errorf("nil Amount: %w", route.ErrInvalidEvent)
	}

	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestWithdrawal()
	if additional == nil || additional.GetPositionInBlock() == nil {
		return fmt.Errorf("nil PositionInBlock: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconBlockExecutionRequestWithdrawalBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockExecutionRequestWithdrawalBatch) appendPayload(event *xatu.DecoratedEvent) {
	withdrawal := event.GetEthV2BeaconBlockExecutionRequestWithdrawal()

	b.SourceAddress.Append([]byte(withdrawal.GetSourceAddress().GetValue()))
	b.ValidatorPubkey.Append(withdrawal.GetValidatorPubkey().GetValue())
	b.Amount.Append(proto.UInt128{Low: withdrawal.GetAmount().GetValue()})
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockExecutionRequestWithdrawalBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestWithdrawal()
	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)

	b.PositionInBlock.Append(uint32(additional.GetPositionInBlock().GetValue()))
}
