package execution

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var mempoolTransactionEventNames = []xatu.Event_Name{
	xatu.Event_MEMPOOL_TRANSACTION,
	xatu.Event_MEMPOOL_TRANSACTION_V2,
}

func init() {
	r, err := route.NewStaticRoute(
		mempoolTransactionTableName,
		mempoolTransactionEventNames,
		func() route.ColumnarBatch { return newmempoolTransactionBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *mempoolTransactionBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	b.appendRuntime(event)
	b.appendMetadata(event)

	if err := b.appendAdditionalData(event); err != nil {
		return fmt.Errorf("appending additional data: %w", err)
	}

	b.rows++

	return nil
}

func (b *mempoolTransactionBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *mempoolTransactionBatch) appendAdditionalData(event *xatu.DecoratedEvent) error {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.appendZeroPayload()

		return nil
	}

	client := event.GetMeta().GetClient()

	// Try V2 additional data first (has more fields).
	if v2 := client.GetMempoolTransactionV2(); v2 != nil {
		return b.appendMempoolTransactionV2(v2)
	}

	// Fall back to V1 additional data.
	if v1 := client.GetMempoolTransaction(); v1 != nil {
		return b.appendMempoolTransactionV1(v1)
	}

	b.appendZeroPayload()

	return nil
}

func (b *mempoolTransactionBatch) appendZeroPayload() {
	b.Hash.Append(nil)
	b.From.Append(nil)
	b.To.Append(proto.Nullable[[]byte]{})
	b.Nonce.Append(0)
	b.GasPrice.Append(proto.UInt128{})
	b.Gas.Append(0)
	b.GasTipCap.Append(proto.Nullable[proto.UInt128]{})
	b.GasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	b.Value.Append(proto.UInt128{})
	b.Type.Append(proto.Nullable[uint8]{})
	b.Size.Append(0)
	b.CallDataSize.Append(0)
	b.BlobGas.Append(proto.Nullable[uint64]{})
	b.BlobGasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	b.BlobHashes.Append(nil)
	b.BlobSidecarsSize.Append(proto.Nullable[uint32]{})
	b.BlobSidecarsEmptySize.Append(proto.Nullable[uint32]{})
}

func (b *mempoolTransactionBatch) appendMempoolTransactionV2(
	v2 *xatu.ClientMeta_AdditionalMempoolTransactionV2Data,
) error {
	b.Hash.Append([]byte(v2.GetHash()))
	b.From.Append([]byte(v2.GetFrom()))

	if to := v2.GetTo(); to != "" {
		b.To.Append(proto.NewNullable[[]byte]([]byte(to)))
	} else {
		b.To.Append(proto.Nullable[[]byte]{})
	}

	if nonce := v2.GetNonce(); nonce != nil {
		b.Nonce.Append(nonce.GetValue())
	} else {
		b.Nonce.Append(0)
	}

	gasPrice, err := route.ParseUInt128(v2.GetGasPrice())
	if err != nil {
		return fmt.Errorf("parsing gas_price: %w", err)
	}

	b.GasPrice.Append(gasPrice)

	if gas := v2.GetGas(); gas != nil {
		b.Gas.Append(gas.GetValue())
	} else {
		b.Gas.Append(0)
	}

	if gtc := v2.GetGasTipCap(); gtc != "" {
		gasTipCap, parseErr := route.ParseUInt128(gtc)
		if parseErr != nil {
			b.GasTipCap.Append(proto.Nullable[proto.UInt128]{})
		} else {
			b.GasTipCap.Append(proto.NewNullable[proto.UInt128](gasTipCap))
		}
	} else {
		b.GasTipCap.Append(proto.Nullable[proto.UInt128]{})
	}

	if gfc := v2.GetGasFeeCap(); gfc != "" {
		gasFeeCap, parseErr := route.ParseUInt128(gfc)
		if parseErr != nil {
			b.GasFeeCap.Append(proto.Nullable[proto.UInt128]{})
		} else {
			b.GasFeeCap.Append(proto.NewNullable[proto.UInt128](gasFeeCap))
		}
	} else {
		b.GasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	}

	value, err := route.ParseUInt128(v2.GetValue())
	if err != nil {
		return fmt.Errorf("parsing value: %w", err)
	}

	b.Value.Append(value)

	if txType := v2.GetType(); txType != nil {
		b.Type.Append(proto.NewNullable[uint8](uint8(txType.GetValue()))) //nolint:gosec // tx type fits uint8
	} else {
		b.Type.Append(proto.Nullable[uint8]{})
	}

	size, _ := parseUint32(v2.GetSize())
	b.Size.Append(size)

	callDataSize, _ := parseUint32(v2.GetCallDataSize())
	b.CallDataSize.Append(callDataSize)

	if blobGas := v2.GetBlobGas(); blobGas != nil {
		b.BlobGas.Append(proto.NewNullable[uint64](blobGas.GetValue()))
	} else {
		b.BlobGas.Append(proto.Nullable[uint64]{})
	}

	if bgfc := v2.GetBlobGasFeeCap(); bgfc != "" {
		blobGasFeeCap, parseErr := route.ParseUInt128(bgfc)
		if parseErr != nil {
			b.BlobGasFeeCap.Append(proto.Nullable[proto.UInt128]{})
		} else {
			b.BlobGasFeeCap.Append(proto.NewNullable[proto.UInt128](blobGasFeeCap))
		}
	} else {
		b.BlobGasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	}

	b.BlobHashes.Append(v2.GetBlobHashes())

	blobSidecarsSize, _ := parseUint32(v2.GetBlobSidecarsSize())
	if v2.GetBlobSidecarsSize() != "" {
		b.BlobSidecarsSize.Append(proto.NewNullable[uint32](blobSidecarsSize))
	} else {
		b.BlobSidecarsSize.Append(proto.Nullable[uint32]{})
	}

	blobSidecarsEmptySize, _ := parseUint32(v2.GetBlobSidecarsEmptySize())
	if v2.GetBlobSidecarsEmptySize() != "" {
		b.BlobSidecarsEmptySize.Append(proto.NewNullable[uint32](blobSidecarsEmptySize))
	} else {
		b.BlobSidecarsEmptySize.Append(proto.Nullable[uint32]{})
	}

	return nil
}

func (b *mempoolTransactionBatch) appendMempoolTransactionV1(
	v1 *xatu.ClientMeta_AdditionalMempoolTransactionData,
) error {
	b.Hash.Append([]byte(v1.GetHash()))
	b.From.Append([]byte(v1.GetFrom()))

	if to := v1.GetTo(); to != "" {
		b.To.Append(proto.NewNullable[[]byte]([]byte(to)))
	} else {
		b.To.Append(proto.Nullable[[]byte]{})
	}

	b.Nonce.Append(v1.GetNonce())

	gasPrice, err := route.ParseUInt128(v1.GetGasPrice())
	if err != nil {
		return fmt.Errorf("parsing gas_price: %w", err)
	}

	b.GasPrice.Append(gasPrice)
	b.Gas.Append(v1.GetGas())
	b.GasTipCap.Append(proto.Nullable[proto.UInt128]{})
	b.GasFeeCap.Append(proto.Nullable[proto.UInt128]{})

	value, err := route.ParseUInt128(v1.GetValue())
	if err != nil {
		return fmt.Errorf("parsing value: %w", err)
	}

	b.Value.Append(value)
	b.Type.Append(proto.Nullable[uint8]{})

	size, _ := parseUint32(v1.GetSize())
	b.Size.Append(size)

	callDataSize, _ := parseUint32(v1.GetCallDataSize())
	b.CallDataSize.Append(callDataSize)

	b.BlobGas.Append(proto.Nullable[uint64]{})
	b.BlobGasFeeCap.Append(proto.Nullable[proto.UInt128]{})
	b.BlobHashes.Append(nil)
	b.BlobSidecarsSize.Append(proto.Nullable[uint32]{})
	b.BlobSidecarsEmptySize.Append(proto.Nullable[uint32]{})

	return nil
}

func parseUint32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(v), nil
}
