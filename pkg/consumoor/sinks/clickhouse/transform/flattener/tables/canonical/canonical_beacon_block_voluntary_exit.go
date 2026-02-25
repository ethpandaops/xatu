package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockVoluntaryExitTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconBlockVoluntaryExitBatch()
		},
	))
}

//nolint:gosec // G115: voluntary exit field values fit uint32.
func (b *canonicalBeaconBlockVoluntaryExitBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV2BeaconBlockVoluntaryExit()
	if payload == nil {
		return fmt.Errorf("nil VoluntaryExit payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockVoluntaryExit()
	blk := addl.GetBlock()

	b.UpdatedDateTime.Append(time.Now())
	b.Slot.Append(uint32(blk.GetSlot().GetNumber().GetValue()))
	b.SlotStartDateTime.Append(blk.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(blk.GetEpoch().GetNumber().GetValue()))
	b.EpochStartDateTime.Append(blk.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(blk.GetRoot()))
	b.BlockVersion.Append(blk.GetVersion())

	msg := payload.GetMessage()
	b.VoluntaryExitMessageEpoch.Append(uint32(msg.GetEpoch().GetValue()))
	b.VoluntaryExitMessageValidatorIndex.Append(uint32(msg.GetValidatorIndex().GetValue()))
	b.VoluntaryExitSignature.Append(payload.GetSignature())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockVoluntaryExitBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockVoluntaryExit()
	if addl == nil {
		return fmt.Errorf("nil VoluntaryExit additional data: %w", flattener.ErrInvalidEvent)
	}

	blk := addl.GetBlock()
	if blk == nil {
		return fmt.Errorf("nil Block identifier: %w", flattener.ErrInvalidEvent)
	}

	if blk.GetEpoch() == nil || blk.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if blk.GetSlot() == nil || blk.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
