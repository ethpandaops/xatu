package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockExecutionRequestConsolidationEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_CONSOLIDATION,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockExecutionRequestConsolidationTableName,
		canonicalBeaconBlockExecutionRequestConsolidationEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockExecutionRequestConsolidationBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockExecutionRequestConsolidationBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockExecutionRequestConsolidation() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_execution_request_consolidation payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockExecutionRequestConsolidationBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockExecutionRequestConsolidationBatch) appendPayload(event *xatu.DecoratedEvent) {
	consolidation := event.GetEthV2BeaconBlockExecutionRequestConsolidation()
	b.SourceAddress.Append([]byte(consolidation.GetSourceAddress().GetValue()))
	b.SourcePubkey.Append(consolidation.GetSourcePubkey().GetValue())
	b.TargetPubkey.Append(consolidation.GetTargetPubkey().GetValue())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockExecutionRequestConsolidationBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestConsolidation()
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
