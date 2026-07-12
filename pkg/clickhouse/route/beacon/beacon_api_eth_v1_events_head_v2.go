package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// The beacon-API head_v2 SSE topic travels as the HEAD_V3 xatu event
// (HEAD_V2 is the schema rev of the v1 head topic, stored in
// beacon_api_eth_v1_events_head).
var beaconApiEthV1EventsHeadV2EventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V3,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsHeadV2TableName,
		beaconApiEthV1EventsHeadV2EventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsHeadV2Batch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsHeadV2Batch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsHeadV3() == nil {
		return fmt.Errorf("nil eth_v1_events_head_v3 payload: %w", route.ErrInvalidEvent)
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

func (b *beaconApiEthV1EventsHeadV2Batch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsHeadV3()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *beaconApiEthV1EventsHeadV2Batch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsHeadV2Batch) appendPayload(event *xatu.DecoratedEvent) {
	head := event.GetEthV1EventsHeadV3()
	if slot := head.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	b.Block.Append([]byte(head.GetBlock()))
	b.PayloadStatus.Append(head.GetPayloadStatus())
	b.EpochTransition.Append(head.GetEpochTransition())
	b.ExecutionOptimistic.Append(head.GetExecutionOptimistic())
	b.CurrentEpochDependentRoot.Append([]byte(head.GetCurrentEpochDependentRoot()))
	b.NextEpochDependentRoot.Append([]byte(head.GetNextEpochDependentRoot()))
}

func (b *beaconApiEthV1EventsHeadV2Batch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsHeadV3())

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
