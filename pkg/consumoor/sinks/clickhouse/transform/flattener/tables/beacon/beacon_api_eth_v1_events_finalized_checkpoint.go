package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsFinalizedCheckpointTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1EventsFinalizedCheckpointBatch()
		},
	))
}

func (b *beaconApiEthV1EventsFinalizedCheckpointBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1EventsFinalizedCheckpointV2()
	if payload == nil {
		return fmt.Errorf("nil EthV1EventsFinalizedCheckpointV2 payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsFinalizedCheckpointV2()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Block.Append([]byte(payload.GetBlock()))
	b.State.Append([]byte(payload.GetState()))
	b.Epoch.Append(uint32(payload.GetEpoch().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.ExecutionOptimistic.Append(false)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsFinalizedCheckpointBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsFinalizedCheckpointV2()

	if payload.GetEpoch() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsFinalizedCheckpointV2()
	if addl == nil {
		return fmt.Errorf("nil EthV1EventsFinalizedCheckpointV2 additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil additional Epoch: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
