package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsFinalizedCheckpointEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsFinalizedCheckpointTableName,
		beaconApiEthV1EventsFinalizedCheckpointEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsFinalizedCheckpointBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsFinalizedCheckpointBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsFinalizedCheckpointV2() == nil {
		return fmt.Errorf("nil eth_v1_events_finalized_checkpoint_v2 payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsFinalizedCheckpointBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsFinalizedCheckpointBatch) appendPayload(event *xatu.DecoratedEvent) {
	checkpointV2 := event.GetEthV1EventsFinalizedCheckpointV2()
	b.Block.Append([]byte(checkpointV2.GetBlock()))
	b.State.Append([]byte(checkpointV2.GetState()))

	if epoch := checkpointV2.GetEpoch(); epoch != nil {
		b.Epoch.Append(uint32(epoch.GetValue())) //nolint:gosec // epoch fits uint32
	} else {
		b.Epoch.Append(0)
	}

	b.ExecutionOptimistic.Append(false)
}

func (b *beaconApiEthV1EventsFinalizedCheckpointBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	additionalV2 := event.GetMeta().GetClient().GetEthV1EventsFinalizedCheckpointV2()
	if additionalV2 == nil {
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	if epoch := additionalV2.GetEpoch(); epoch != nil {
		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.EpochStartDateTime.Append(time.Time{})
	}
}
