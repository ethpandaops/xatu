package mev

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		mevRelayProposerPayloadDeliveredTableName,
		[]xatu.Event_Name{xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED},
		func() flattener.ColumnarBatch {
			return newmevRelayProposerPayloadDeliveredBatch()
		},
	))
}

func (b *mevRelayProposerPayloadDeliveredBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetMevRelayPayloadDelivered()
	if payload == nil {
		return fmt.Errorf("nil MevRelayPayloadDelivered payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetMevRelayPayloadDelivered()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(addl.GetSlot().GetNumber().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.WallclockSlot.Append(uint32(addl.GetWallclockSlot().GetNumber().GetValue())) //nolint:gosec // G115: wallclock slot fits uint32.
	b.WallclockSlotStartDateTime.Append(addl.GetWallclockSlot().GetStartDateTime().AsTime())
	b.WallclockEpoch.Append(uint32(addl.GetWallclockEpoch().GetNumber().GetValue())) //nolint:gosec // G115: wallclock epoch fits uint32.
	b.WallclockEpochStartDateTime.Append(addl.GetWallclockEpoch().GetStartDateTime().AsTime())
	b.BlockNumber.Append(payload.GetBlockNumber().GetValue())
	b.RelayName.Append(addl.GetRelay().GetName().GetValue())
	b.BlockHash.Append([]byte(payload.GetBlockHash().GetValue()))
	b.ProposerPubkey.Append(payload.GetProposerPubkey().GetValue())
	b.BuilderPubkey.Append(payload.GetBuilderPubkey().GetValue())
	b.ProposerFeeRecipient.Append([]byte(payload.GetProposerFeeRecipient().GetValue()))
	b.GasLimit.Append(payload.GetGasLimit().GetValue())
	b.GasUsed.Append(payload.GetGasUsed().GetValue())
	b.Value.Append(flattener.ParseUInt256(payload.GetValue().GetValue()))
	b.NumTx.Append(uint32(payload.GetNumTx().GetValue())) //nolint:gosec // G115: num_tx fits uint32.

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *mevRelayProposerPayloadDeliveredBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetMevRelayPayloadDelivered()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetMevRelayPayloadDelivered()
	if addl == nil {
		return fmt.Errorf("nil MevRelayPayloadDelivered additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil || addl.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
