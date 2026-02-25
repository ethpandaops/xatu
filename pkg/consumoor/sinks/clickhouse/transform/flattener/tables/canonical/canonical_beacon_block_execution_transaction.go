package canonical

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockExecutionTransactionTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconBlockExecutionTransactionBatch()
		},
	))
}

//nolint:gosec // G115: transaction field values fit their target types.
func (b *canonicalBeaconBlockExecutionTransactionBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV2BeaconBlockExecutionTransaction()
	if payload == nil {
		return fmt.Errorf("nil ExecutionTransaction payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionTransaction()
	blk := addl.GetBlock()

	b.UpdatedDateTime.Append(time.Now())
	b.Slot.Append(uint32(blk.GetSlot().GetNumber().GetValue()))
	b.SlotStartDateTime.Append(blk.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(blk.GetEpoch().GetNumber().GetValue()))
	b.EpochStartDateTime.Append(blk.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(blk.GetRoot()))
	b.BlockVersion.Append(blk.GetVersion())

	b.Position.Append(uint32(addl.GetPositionInBlock().GetValue()))
	b.Hash.Append([]byte(payload.GetHash()))
	b.From.Append([]byte(payload.GetFrom()))

	if payload.GetTo() != "" {
		b.To.Append(proto.NewNullable[[]byte]([]byte(payload.GetTo())))
	} else {
		b.To.Append(proto.Nullable[[]byte]{})
	}

	b.Nonce.Append(payload.GetNonce().GetValue())
	b.GasPrice.Append(flattener.ParseUInt128(payload.GetGasPrice()))
	b.Gas.Append(payload.GetGas().GetValue())

	if payload.GetGasTipCap() != "" {
		b.GasTipCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetGasTipCap())))
	} else {
		b.GasTipCap.Append(proto.Nullable[proto.UInt128]{})
	}

	if payload.GetGasFeeCap() != "" {
		b.GasFeeCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetGasFeeCap())))
	} else {
		b.GasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	}

	b.Value.Append(flattener.ParseUInt128(payload.GetValue()))
	b.Type.Append(uint8(payload.GetType().GetValue()))

	// Size and CallDataSize come from additional data as strings.
	b.Size.Append(parseUint32(addl.GetSize()))
	b.CallDataSize.Append(parseUint32(addl.GetCallDataSize()))

	if payload.GetBlobGas() != nil {
		b.BlobGas.Append(proto.NewNullable[uint64](payload.GetBlobGas().GetValue()))
	} else {
		b.BlobGas.Append(proto.Nullable[uint64]{})
	}

	if payload.GetBlobGasFeeCap() != "" {
		b.BlobGasFeeCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBlobGasFeeCap())))
	} else {
		b.BlobGasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	}

	b.BlobHashes.Append(payload.GetBlobHashes())

	if addl.GetBlobSidecarsSize() != "" {
		b.BlobSidecarsSize.Append(proto.NewNullable[uint32](parseUint32(addl.GetBlobSidecarsSize())))
	} else {
		b.BlobSidecarsSize.Append(proto.Nullable[uint32]{})
	}

	if addl.GetBlobSidecarsEmptySize() != "" {
		b.BlobSidecarsEmptySize.Append(proto.NewNullable[uint32](parseUint32(addl.GetBlobSidecarsEmptySize())))
	} else {
		b.BlobSidecarsEmptySize.Append(proto.Nullable[uint32]{})
	}

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockExecutionTransactionBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionTransaction()
	if addl == nil {
		return fmt.Errorf("nil ExecutionTransaction additional data: %w", flattener.ErrInvalidEvent)
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

// parseUint32 parses a decimal string to uint32, returning 0 on failure.
func parseUint32(s string) uint32 {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0
	}

	return uint32(v)
}
