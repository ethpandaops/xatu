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
		beaconApiEthV1EventsDataColumnSidecarTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1EventsDataColumnSidecarBatch()
		},
	))
}

func (b *beaconApiEthV1EventsDataColumnSidecarBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1EventsDataColumnSidecar()
	if payload == nil {
		return fmt.Errorf("nil EthV1EventsDataColumnSidecar payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsDataColumnSidecar()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.PropagationSlotStartDiff.Append(uint32(addl.GetPropagation().GetSlotStartDiff().GetValue())) //nolint:gosec // G115: propagation diff fits uint32.
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue()))                                 //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(payload.GetBlockRoot()))
	b.ColumnIndex.Append(payload.GetIndex().GetValue())
	b.KzgCommitmentsCount.Append(payload.GetKzgCommitmentsCount().GetValue())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsDataColumnSidecarBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsDataColumnSidecar()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetIndex() == nil {
		return fmt.Errorf("nil Index: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetKzgCommitmentsCount() == nil {
		return fmt.Errorf("nil KzgCommitmentsCount: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsDataColumnSidecar()
	if addl == nil {
		return fmt.Errorf("nil EthV1EventsDataColumnSidecar additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetPropagation() == nil || addl.GetPropagation().GetSlotStartDiff() == nil {
		return fmt.Errorf("nil PropagationSlotStartDiff: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
