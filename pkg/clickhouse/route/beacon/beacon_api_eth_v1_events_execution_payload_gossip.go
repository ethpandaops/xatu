package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsExecutionPayloadGossipEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_GOSSIP,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsExecutionPayloadGossipTableName,
		beaconApiEthV1EventsExecutionPayloadGossipEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsExecutionPayloadGossipBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsExecutionPayloadGossipBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsExecutionPayloadGossip() == nil {
		return fmt.Errorf("nil eth_v1_events_execution_payload_gossip payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsExecutionPayloadGossipBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsExecutionPayloadGossipBatch) appendPayload(event *xatu.DecoratedEvent) {
	envelope := event.GetEthV1EventsExecutionPayloadGossip()

	msg := envelope.GetMessage()

	b.BlockRoot.Append([]byte(msg.GetBeaconBlockRoot()))

	if builderIndex := msg.GetBuilderIndex(); builderIndex != nil {
		b.BuilderIndex.Append(builderIndex.GetValue())
	} else {
		b.BuilderIndex.Append(0)
	}

	if payload := msg.GetPayload(); payload != nil {
		b.BlockHash.Append([]byte(payload.GetBlockHash()))
		b.StateRoot.Append([]byte(payload.GetStateRoot()))
	} else {
		b.BlockHash.Append(nil)
		b.StateRoot.Append(nil)
	}
}

func (b *beaconApiEthV1EventsExecutionPayloadGossipBatch) appendAdditionalData(
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
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsExecutionPayloadGossip())

	b.Slot.Append(uint32(additional.Slot)) //nolint:gosec // slot fits uint32
	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
