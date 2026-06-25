package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStatePendingConsolidationEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStatePendingConsolidationTableName,
		canonicalBeaconStatePendingConsolidationEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStatePendingConsolidationBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStatePendingConsolidationBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStatePendingConsolidation()
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

func (b *canonicalBeaconStatePendingConsolidationBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconStatePendingConsolidation()

	if payload.GetSourceIndex() == nil {
		return fmt.Errorf("nil SourceIndex: %w", route.ErrInvalidEvent)
	}

	if payload.GetTargetIndex() == nil {
		return fmt.Errorf("nil TargetIndex: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingConsolidation()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	if extra.GetPositionInQueue() == nil {
		return fmt.Errorf("nil PositionInQueue: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconStatePendingConsolidationBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStatePendingConsolidationBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconStatePendingConsolidation()

	b.SourceIndex.Append(uint32(payload.GetSourceIndex().GetValue())) //nolint:gosec // bounded by uint32 column
	b.TargetIndex.Append(uint32(payload.GetTargetIndex().GetValue())) //nolint:gosec // bounded by uint32 column
}

func (b *canonicalBeaconStatePendingConsolidationBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingConsolidation()
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
