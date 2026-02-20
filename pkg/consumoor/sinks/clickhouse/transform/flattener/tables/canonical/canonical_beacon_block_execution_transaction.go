package canonical

import (
	"strconv"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockExecutionTransactionEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockExecutionTransactionTableName,
		canonicalBeaconBlockExecutionTransactionEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconBlockExecutionTransactionBatch() },
	))
}

func (b *canonicalBeaconBlockExecutionTransactionBatch) FlattenTo(
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

func (b *canonicalBeaconBlockExecutionTransactionBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockExecutionTransactionBatch) appendPayload(event *xatu.DecoratedEvent) {
	tx := event.GetEthV2BeaconBlockExecutionTransaction()
	if tx == nil {
		b.Hash.Append(nil)
		b.From.Append(nil)
		b.To.Append(proto.Nullable[[]byte]{})
		b.Nonce.Append(0)
		b.GasPrice.Append(flattener.ParseUInt128("0"))
		b.Gas.Append(0)
		b.GasTipCap.Append(proto.Nullable[proto.UInt128]{})
		b.GasFeeCap.Append(proto.Nullable[proto.UInt128]{})
		b.Value.Append(flattener.ParseUInt128("0"))
		b.Type.Append(0)
		b.BlobGas.Append(proto.Nullable[uint64]{})
		b.BlobGasFeeCap.Append(proto.Nullable[proto.UInt128]{})
		b.BlobHashes.Append([]string{})

		return
	}

	b.Hash.Append([]byte(tx.GetHash()))
	b.From.Append([]byte(tx.GetFrom()))

	toStr := tx.GetTo()
	if toStr != "" {
		b.To.Append(proto.NewNullable[[]byte]([]byte(toStr)))
	} else {
		b.To.Append(proto.Nullable[[]byte]{})
	}

	b.GasPrice.Append(flattener.ParseUInt128(tx.GetGasPrice()))

	gasTipCap := tx.GetGasTipCap()
	if gasTipCap != "" {
		b.GasTipCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(gasTipCap)))
	} else {
		b.GasTipCap.Append(proto.Nullable[proto.UInt128]{})
	}

	gasFeeCap := tx.GetGasFeeCap()
	if gasFeeCap != "" {
		b.GasFeeCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(gasFeeCap)))
	} else {
		b.GasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	}

	b.Value.Append(flattener.ParseUInt128(tx.GetValue()))

	blobGasFeeCap := tx.GetBlobGasFeeCap()
	if blobGasFeeCap != "" {
		b.BlobGasFeeCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(blobGasFeeCap)))
	} else {
		b.BlobGasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	}

	b.BlobHashes.Append(tx.GetBlobHashes())

	if nonce := tx.GetNonce(); nonce != nil {
		b.Nonce.Append(nonce.GetValue())
	} else {
		b.Nonce.Append(0)
	}

	if gas := tx.GetGas(); gas != nil {
		b.Gas.Append(gas.GetValue())
	} else {
		b.Gas.Append(0)
	}

	if txType := tx.GetType(); txType != nil {
		b.Type.Append(uint8(txType.GetValue())) //nolint:gosec // G115
	} else {
		b.Type.Append(0)
	}

	if blobGas := tx.GetBlobGas(); blobGas != nil {
		b.BlobGas.Append(proto.NewNullable[uint64](blobGas.GetValue()))
	} else {
		b.BlobGas.Append(proto.Nullable[uint64]{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockExecutionTransactionBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionTransaction()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockVersion.Append("")
		b.BlockRoot.Append(nil)
		b.Position.Append(0)
		b.Size.Append(0)
		b.CallDataSize.Append(0)
		b.BlobSidecarsSize.Append(proto.Nullable[uint32]{})
		b.BlobSidecarsEmptySize.Append(proto.Nullable[uint32]{})

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)

	if positionInBlock := additional.GetPositionInBlock(); positionInBlock != nil {
		b.Position.Append(uint32(positionInBlock.GetValue()))
	} else {
		b.Position.Append(0)
	}

	if sizeStr := additional.GetSize(); sizeStr != "" {
		if parsed, err := strconv.ParseUint(sizeStr, 10, 32); err == nil {
			b.Size.Append(uint32(parsed))
		} else {
			b.Size.Append(0)
		}
	} else {
		b.Size.Append(0)
	}

	if callDataSizeStr := additional.GetCallDataSize(); callDataSizeStr != "" {
		if parsed, err := strconv.ParseUint(callDataSizeStr, 10, 32); err == nil {
			b.CallDataSize.Append(uint32(parsed))
		} else {
			b.CallDataSize.Append(0)
		}
	} else {
		b.CallDataSize.Append(0)
	}

	if blobSidecarsSizeStr := additional.GetBlobSidecarsSize(); blobSidecarsSizeStr != "" {
		if parsed, err := strconv.ParseUint(blobSidecarsSizeStr, 10, 32); err == nil {
			b.BlobSidecarsSize.Append(proto.NewNullable[uint32](uint32(parsed)))
		} else {
			b.BlobSidecarsSize.Append(proto.Nullable[uint32]{})
		}
	} else {
		b.BlobSidecarsSize.Append(proto.Nullable[uint32]{})
	}

	if blobSidecarsEmptySizeStr := additional.GetBlobSidecarsEmptySize(); blobSidecarsEmptySizeStr != "" {
		if parsed, err := strconv.ParseUint(blobSidecarsEmptySizeStr, 10, 32); err == nil {
			b.BlobSidecarsEmptySize.Append(proto.NewNullable[uint32](uint32(parsed)))
		} else {
			b.BlobSidecarsEmptySize.Append(proto.Nullable[uint32]{})
		}
	} else {
		b.BlobSidecarsEmptySize.Append(proto.Nullable[uint32]{})
	}
}
