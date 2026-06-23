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

	extra := event.GetMeta().GetClient().GetEthV1BeaconStatePendingConsolidation()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	var (
		epoch          uint32
		epochStartTime time.Time
	)

	if number := extra.GetEpoch().GetNumber(); number != nil {
		epoch = uint32(number.GetValue()) //nolint:gosec // bounded by uint32 column
	}

	if start := extra.GetEpoch().GetStartDateTime(); start != nil {
		epochStartTime = start.AsTime()
	}

	var positionInQueue uint32
	if pos := extra.GetPositionInQueue(); pos != nil {
		positionInQueue = uint32(pos.GetValue()) //nolint:gosec // bounded by uint32 column
	}

	var sourceIndex uint32
	if src := payload.GetSourceIndex(); src != nil {
		sourceIndex = uint32(src.GetValue()) //nolint:gosec // bounded by uint32 column
	}

	var targetIndex uint32
	if tgt := payload.GetTargetIndex(); tgt != nil {
		targetIndex = uint32(tgt.GetValue()) //nolint:gosec // bounded by uint32 column
	}

	b.UpdatedDateTime.Append(time.Now())
	b.Epoch.Append(epoch)
	b.EpochStartDateTime.Append(epochStartTime)
	b.StateID.Append(extra.GetStateId())
	b.PositionInQueue.Append(positionInQueue)
	b.SourceIndex.Append(sourceIndex)
	b.TargetIndex.Append(targetIndex)
	b.appendMetadata(event)
	b.rows++

	return nil
}
