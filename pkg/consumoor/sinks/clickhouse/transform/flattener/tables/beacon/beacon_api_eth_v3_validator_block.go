package beacon

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV3ValidatorBlockTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV3ValidatorBlockBatch()
		},
	))
}

func (b *beaconApiEthV3ValidatorBlockBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV3ValidatorBlock()
	if payload == nil {
		return fmt.Errorf("nil EthV3ValidatorBlock payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV3ValidatorBlock()

	// Extract fork-specific block fields.
	f := extractV2BlockFields(payload)

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(f.Slot)) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.BlockVersion.Append(f.BlockVersion)

	// Nullable block-level additional data.
	if addl.GetTotalBytes() != nil {
		b.BlockTotalBytes.Append(proto.NewNullable[uint32](uint32(addl.GetTotalBytes().GetValue()))) //nolint:gosec // G115: total bytes fits uint32.
	} else {
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if addl.GetTotalBytesCompressed() != nil {
		b.BlockTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(addl.GetTotalBytesCompressed().GetValue()))) //nolint:gosec // G115: total bytes compressed fits uint32.
	} else {
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}

	// Consensus/execution payload values from additional data (string -> uint64 nullable).
	appendNullableUInt64FromString(b.ConsensusPayloadValue, addl.GetConsensusValue())
	appendNullableUInt64FromString(b.ExecutionPayloadValue, addl.GetExecutionValue())

	b.ExecutionPayloadBlockNumber.Append(uint32(f.EPBlockNumber)) //nolint:gosec // G115: block number fits uint32.

	if f.EPBaseFeePerGas != "" {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(f.EPBaseFeePerGas)))
	} else {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	}

	appendNullableUInt64(b.ExecutionPayloadBlobGasUsed, f.EPBlobGasUsed)
	appendNullableUInt64(b.ExecutionPayloadExcessBlobGas, f.EPExcessBlobGas)
	appendNullableUInt64(b.ExecutionPayloadGasLimit, f.EPGasLimit)
	appendNullableUInt64(b.ExecutionPayloadGasUsed, f.EPGasUsed)

	if addl.GetTransactionsCount() != nil {
		b.ExecutionPayloadTransactionsCount.Append(proto.NewNullable[uint32](uint32(addl.GetTransactionsCount().GetValue()))) //nolint:gosec // G115: tx count fits uint32.
	} else {
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
	}

	if addl.GetTransactionsTotalBytes() != nil {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.NewNullable[uint32](uint32(addl.GetTransactionsTotalBytes().GetValue()))) //nolint:gosec // G115: tx total bytes fits uint32.
	} else {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if addl.GetTransactionsTotalBytesCompressed() != nil {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(addl.GetTransactionsTotalBytesCompressed().GetValue()))) //nolint:gosec // G115: tx total bytes compressed fits uint32.
	} else {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV3ValidatorBlockBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV3ValidatorBlock()

	if payload.GetMessage() == nil {
		return fmt.Errorf("nil block message (no fork variant): %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV3ValidatorBlock()
	if addl == nil {
		return fmt.Errorf("nil EthV3ValidatorBlock additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}

// appendNullableUInt64FromString parses a decimal string as uint64 and
// appends it as a nullable column entry. An empty string appends an unset
// nullable.
func appendNullableUInt64FromString(col *proto.ColNullable[uint64], s string) {
	if s != "" {
		// Parse via UInt128; use the low 64 bits.
		v := flattener.ParseUInt128(s)
		col.Append(proto.NewNullable[uint64](v.Low))
	} else {
		col.Append(proto.Nullable[uint64]{})
	}
}
