package beacon

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsDataColumnSidecarEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsDataColumnSidecarTableName,
		beaconApiEthV1EventsDataColumnSidecarEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1EventsDataColumnSidecarBatch() },
	))
}

func (b *beaconApiEthV1EventsDataColumnSidecarBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsDataColumnSidecarBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsDataColumnSidecarBatch) appendPayload(event *xatu.DecoratedEvent) {
	sidecar := event.GetEthV1EventsDataColumnSidecar()
	if sidecar == nil {
		b.Slot.Append(0)
		b.BlockRoot.Append(nil)
		b.ColumnIndex.Append(0)
		b.KzgCommitmentsCount.Append(0)

		return
	}

	if slot := sidecar.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	b.BlockRoot.Append([]byte(sidecar.GetBlockRoot()))

	if index := sidecar.GetIndex(); index != nil {
		b.ColumnIndex.Append(index.GetValue())
	} else {
		b.ColumnIndex.Append(0)
	}

	if kzgCommitmentsCount := sidecar.GetKzgCommitmentsCount(); kzgCommitmentsCount != nil {
		b.KzgCommitmentsCount.Append(kzgCommitmentsCount.GetValue())
	} else {
		b.KzgCommitmentsCount.Append(0)
	}
}

func (b *beaconApiEthV1EventsDataColumnSidecarBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsDataColumnSidecar())

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
