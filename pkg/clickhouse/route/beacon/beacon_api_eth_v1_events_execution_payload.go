package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsExecutionPayloadEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsExecutionPayloadTableName,
		beaconApiEthV1EventsExecutionPayloadEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsExecutionPayloadBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsExecutionPayloadBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsExecutionPayload() == nil {
		return fmt.Errorf("nil eth_v1_events_execution_payload payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsExecutionPayloadBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsExecutionPayloadBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1EventsExecutionPayload()

	b.BlockRoot.Append([]byte(payload.GetBlockRoot()))
	b.BlockHash.Append([]byte(payload.GetBlockHash()))

	if builderIndex := payload.GetBuilderIndex(); builderIndex != nil {
		b.BuilderIndex.Append(builderIndex.GetValue())
	} else {
		b.BuilderIndex.Append(0)
	}

	b.ExecutionOptimistic.Append(payload.GetExecutionOptimistic())
}

func (b *beaconApiEthV1EventsExecutionPayloadBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsExecutionPayload())

	b.Slot.Append(uint32(additional.Slot)) //nolint:gosec // slot fits uint32
	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
