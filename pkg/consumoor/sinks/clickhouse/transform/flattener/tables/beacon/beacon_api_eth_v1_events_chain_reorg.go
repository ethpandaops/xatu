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
		beaconApiEthV1EventsChainReorgTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1EventsChainReorgBatch()
		},
	))
}

func (b *beaconApiEthV1EventsChainReorgBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1EventsChainReorgV2()
	if payload == nil {
		return fmt.Errorf("nil EthV1EventsChainReorgV2 payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsChainReorgV2()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.PropagationSlotStartDiff.Append(uint32(addl.GetPropagation().GetSlotStartDiff().GetValue())) //nolint:gosec // G115: propagation diff fits uint32.
	b.Depth.Append(uint16(payload.GetDepth().GetValue()))                                          //nolint:gosec // G115: reorg depth fits uint16.
	b.OldHeadBlock.Append([]byte(payload.GetOldHeadBlock()))
	b.NewHeadBlock.Append([]byte(payload.GetNewHeadBlock()))
	b.OldHeadState.Append([]byte(payload.GetOldHeadState()))
	b.NewHeadState.Append([]byte(payload.GetNewHeadState()))
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.ExecutionOptimistic.Append(false)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsChainReorgBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsChainReorgV2()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetDepth() == nil {
		return fmt.Errorf("nil Depth: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetEpoch() == nil {
		return fmt.Errorf("nil payload Epoch: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsChainReorgV2()
	if addl == nil {
		return fmt.Errorf("nil EthV1EventsChainReorgV2 additional data: %w", flattener.ErrInvalidEvent)
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
