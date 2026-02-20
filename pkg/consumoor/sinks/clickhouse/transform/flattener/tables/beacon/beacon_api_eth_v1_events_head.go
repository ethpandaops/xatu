package beacon

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsHeadEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsHeadTableName,
		beaconApiEthV1EventsHeadEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1EventsHeadBatch() },
	))
}

func (b *beaconApiEthV1EventsHeadBatch) FlattenTo(
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

func (b *beaconApiEthV1EventsHeadBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsHeadBatch) appendPayload(event *xatu.DecoratedEvent) {
	headV2 := event.GetEthV1EventsHeadV2()
	if headV2 == nil {
		b.Slot.Append(0)
		b.Block.Append(nil)
		b.EpochTransition.Append(false)
		b.ExecutionOptimistic.Append(false)
		b.PreviousDutyDependentRoot.Append(nil)
		b.CurrentDutyDependentRoot.Append(nil)

		return
	}

	if slot := headV2.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	b.Block.Append([]byte(headV2.GetBlock()))
	b.EpochTransition.Append(headV2.GetEpochTransition())
	b.ExecutionOptimistic.Append(false)
	b.PreviousDutyDependentRoot.Append([]byte(headV2.GetPreviousDutyDependentRoot()))
	b.CurrentDutyDependentRoot.Append([]byte(headV2.GetCurrentDutyDependentRoot()))
}

func (b *beaconApiEthV1EventsHeadBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsHeadV2())

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
