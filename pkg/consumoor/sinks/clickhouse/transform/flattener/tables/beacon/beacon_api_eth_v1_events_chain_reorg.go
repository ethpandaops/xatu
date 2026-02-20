package beacon

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsChainReorgEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsChainReorgTableName,
		beaconApiEthV1EventsChainReorgEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1EventsChainReorgBatch() },
	))
}

func (b *beaconApiEthV1EventsChainReorgBatch) FlattenTo(
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

func (b *beaconApiEthV1EventsChainReorgBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsChainReorgBatch) appendPayload(event *xatu.DecoratedEvent) {
	chainReorgV2 := event.GetEthV1EventsChainReorgV2()
	if chainReorgV2 == nil {
		b.Slot.Append(0)
		b.Depth.Append(0)
		b.OldHeadBlock.Append(nil)
		b.NewHeadBlock.Append(nil)
		b.OldHeadState.Append(nil)
		b.NewHeadState.Append(nil)
		b.Epoch.Append(0)
		b.ExecutionOptimistic.Append(false)

		return
	}

	if slot := chainReorgV2.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	if depth := chainReorgV2.GetDepth(); depth != nil {
		b.Depth.Append(uint16(depth.GetValue())) //nolint:gosec // depth fits uint16
	} else {
		b.Depth.Append(0)
	}

	b.OldHeadBlock.Append([]byte(chainReorgV2.GetOldHeadBlock()))
	b.NewHeadBlock.Append([]byte(chainReorgV2.GetNewHeadBlock()))
	b.OldHeadState.Append([]byte(chainReorgV2.GetOldHeadState()))
	b.NewHeadState.Append([]byte(chainReorgV2.GetNewHeadState()))

	if epoch := chainReorgV2.GetEpoch(); epoch != nil {
		b.Epoch.Append(uint32(epoch.GetValue())) //nolint:gosec // epoch fits uint32
	} else {
		b.Epoch.Append(0)
	}

	b.ExecutionOptimistic.Append(false)
}

func (b *beaconApiEthV1EventsChainReorgBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsChainReorgV2())

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
