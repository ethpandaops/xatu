package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsBlockGossipEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsBlockGossipTableName,
		beaconApiEthV1EventsBlockGossipEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsBlockGossipBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsBlockGossipBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsBlockGossip() == nil {
		return fmt.Errorf("nil eth_v1_events_block_gossip payload: %w", route.ErrInvalidEvent)
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

func (b *beaconApiEthV1EventsBlockGossipBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsBlockGossip()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *beaconApiEthV1EventsBlockGossipBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsBlockGossipBatch) appendPayload(event *xatu.DecoratedEvent) {
	gossip := event.GetEthV1EventsBlockGossip()
	if slot := gossip.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	b.Block.Append([]byte(gossip.GetBlock()))
}

func (b *beaconApiEthV1EventsBlockGossipBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsBlockGossip())

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
