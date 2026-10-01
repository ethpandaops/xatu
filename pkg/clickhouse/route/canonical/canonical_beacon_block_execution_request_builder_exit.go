package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockExecutionRequestBuilderExitEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_EXIT,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockExecutionRequestBuilderExitTableName,
		canonicalBeaconBlockExecutionRequestBuilderExitEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockExecutionRequestBuilderExitBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockExecutionRequestBuilderExitBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockExecutionRequestBuilderExit() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_execution_request_builder_exit payload: %w", route.ErrInvalidEvent)
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

func (b *canonicalBeaconBlockExecutionRequestBuilderExitBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV2BeaconBlockExecutionRequestBuilderExit()

	if payload.GetSourceAddress() == nil {
		return fmt.Errorf("nil SourceAddress: %w", route.ErrInvalidEvent)
	}

	if payload.GetPubkey() == nil {
		return fmt.Errorf("nil Pubkey: %w", route.ErrInvalidEvent)
	}

	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderExit()
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

func (b *canonicalBeaconBlockExecutionRequestBuilderExitBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockExecutionRequestBuilderExitBatch) appendPayload(event *xatu.DecoratedEvent) {
	exit := event.GetEthV2BeaconBlockExecutionRequestBuilderExit()

	b.SourceAddress.Append([]byte(exit.GetSourceAddress().GetValue()))
	b.Pubkey.Append(exit.GetPubkey().GetValue())
}

//nolint:gosec // G115: proto uint64 position is bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockExecutionRequestBuilderExitBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderExit()
	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)

	b.BlockNumber.Append(additional.GetBlockNumber().GetValue())
	b.BlockHash.Append([]byte(additional.GetBlockHash()))
	b.PositionInBlock.Append(uint32(additional.GetPositionInBlock().GetValue()))
}
