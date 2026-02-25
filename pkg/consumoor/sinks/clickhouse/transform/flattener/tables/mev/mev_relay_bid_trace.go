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
		mevRelayBidTraceTableName,
		[]xatu.Event_Name{xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION},
		func() flattener.ColumnarBatch {
			return newmevRelayBidTraceBatch()
		},
	))
}

func (b *mevRelayBidTraceBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetMevRelayBidTraceBuilderBlockSubmission()
	if payload == nil {
		return fmt.Errorf("nil MevRelayBidTraceBuilderBlockSubmission payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetMevRelayBidTraceBuilderBlockSubmission()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(addl.GetSlot().GetNumber().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.WallclockRequestSlot.Append(uint32(addl.GetWallclockSlot().GetNumber().GetValue())) //nolint:gosec // G115: wallclock slot fits uint32.
	b.WallclockRequestSlotStartDateTime.Append(addl.GetWallclockSlot().GetStartDateTime().AsTime())
	b.WallclockRequestEpoch.Append(uint32(addl.GetWallclockEpoch().GetNumber().GetValue())) //nolint:gosec // G115: wallclock epoch fits uint32.
	b.WallclockRequestEpochStartDateTime.Append(addl.GetWallclockEpoch().GetStartDateTime().AsTime())
	b.RequestedAtSlotTime.Append(uint32(addl.GetRequestedAtSlotTime().GetValue())) //nolint:gosec // G115: slot time fits uint32.
	b.ResponseAtSlotTime.Append(uint32(addl.GetResponseAtSlotTime().GetValue()))   //nolint:gosec // G115: slot time fits uint32.
	b.RelayName.Append(addl.GetRelay().GetName().GetValue())
	b.ParentHash.Append([]byte(payload.GetParentHash().GetValue()))
	b.BlockNumber.Append(payload.GetBlockNumber().GetValue())
	b.BlockHash.Append([]byte(payload.GetBlockHash().GetValue()))
	b.BuilderPubkey.Append(payload.GetBuilderPubkey().GetValue())
	b.ProposerPubkey.Append(payload.GetProposerPubkey().GetValue())
	b.ProposerFeeRecipient.Append([]byte(payload.GetProposerFeeRecipient().GetValue()))
	b.GasLimit.Append(payload.GetGasLimit().GetValue())
	b.GasUsed.Append(payload.GetGasUsed().GetValue())
	b.Value.Append(flattener.ParseUInt256(payload.GetValue().GetValue()))
	b.NumTx.Append(uint32(payload.GetNumTx().GetValue())) //nolint:gosec // G115: num_tx fits uint32.
	b.Timestamp.Append(payload.GetTimestamp().GetValue())
	b.TimestampMs.Append(payload.GetTimestampMs().GetValue())
	b.OptimisticSubmission.Append(payload.GetOptimisticSubmission().GetValue())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *mevRelayBidTraceBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetMevRelayBidTraceBuilderBlockSubmission()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetOptimisticSubmission() == nil {
		return fmt.Errorf("nil OptimisticSubmission: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetMevRelayBidTraceBuilderBlockSubmission()
	if addl == nil {
		return fmt.Errorf("nil MevRelayBidTraceBuilderBlockSubmission additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil || addl.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
