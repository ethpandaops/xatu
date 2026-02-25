package execution

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
		mempoolTransactionTableName,
		[]xatu.Event_Name{xatu.Event_MEMPOOL_TRANSACTION_V2},
		func() flattener.ColumnarBatch {
			return newmempoolTransactionBatch()
		},
	))
}

func (b *mempoolTransactionBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	addl := event.GetMeta().GetClient().GetMempoolTransactionV2()
	if addl == nil {
		return fmt.Errorf("nil MempoolTransactionV2 additional data: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(addl); err != nil {
		return err
	}

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Hash.Append([]byte(addl.GetHash()))
	b.From.Append([]byte(addl.GetFrom()))

	if v := addl.GetTo(); v != "" {
		b.To.Append(proto.NewNullable[[]byte]([]byte(v)))
	} else {
		b.To.Append(proto.Nullable[[]byte]{})
	}

	b.Nonce.Append(addl.GetNonce().GetValue())
	b.GasPrice.Append(flattener.ParseUInt128(addl.GetGasPrice()))
	b.Gas.Append(addl.GetGas().GetValue())

	if v := addl.GetGasTipCap(); v != "" {
		b.GasTipCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(v)))
	} else {
		b.GasTipCap.Append(proto.Nullable[proto.UInt128]{})
	}

	if v := addl.GetGasFeeCap(); v != "" {
		b.GasFeeCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(v)))
	} else {
		b.GasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	}

	b.Value.Append(flattener.ParseUInt128(addl.GetValue()))

	if t := addl.GetType(); t != nil {
		b.Type.Append(proto.NewNullable[uint8](uint8(t.GetValue()))) //nolint:gosec // G115: tx type fits uint8.
	} else {
		b.Type.Append(proto.Nullable[uint8]{})
	}

	size, _ := strconv.ParseUint(addl.GetSize(), 10, 32)
	b.Size.Append(uint32(size))

	callDataSize, _ := strconv.ParseUint(addl.GetCallDataSize(), 10, 32)
	b.CallDataSize.Append(uint32(callDataSize))

	if bg := addl.GetBlobGas(); bg != nil {
		b.BlobGas.Append(proto.NewNullable[uint64](bg.GetValue()))
	} else {
		b.BlobGas.Append(proto.Nullable[uint64]{})
	}

	if v := addl.GetBlobGasFeeCap(); v != "" {
		b.BlobGasFeeCap.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(v)))
	} else {
		b.BlobGasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	}

	b.BlobHashes.Append(addl.GetBlobHashes())

	if v := addl.GetBlobSidecarsSize(); v != "" {
		parsed, _ := strconv.ParseUint(v, 10, 32)
		b.BlobSidecarsSize.Append(proto.NewNullable[uint32](uint32(parsed)))
	} else {
		b.BlobSidecarsSize.Append(proto.Nullable[uint32]{})
	}

	if v := addl.GetBlobSidecarsEmptySize(); v != "" {
		parsed, _ := strconv.ParseUint(v, 10, 32)
		b.BlobSidecarsEmptySize.Append(proto.NewNullable[uint32](uint32(parsed)))
	} else {
		b.BlobSidecarsEmptySize.Append(proto.Nullable[uint32]{})
	}

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *mempoolTransactionBatch) validate(
	addl *xatu.ClientMeta_AdditionalMempoolTransactionV2Data,
) error {
	if addl.GetHash() == "" {
		return fmt.Errorf("nil Hash: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetNonce() == nil {
		return fmt.Errorf("nil Nonce: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetGas() == nil {
		return fmt.Errorf("nil Gas: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetGasPrice() == "" {
		return fmt.Errorf("nil GasPrice: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetValue() == "" {
		return fmt.Errorf("nil Value: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
