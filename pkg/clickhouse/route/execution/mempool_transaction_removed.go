package execution

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var mempoolTransactionRemovedEventNames = []xatu.Event_Name{
	xatu.Event_MEMPOOL_TRANSACTION_REMOVED,
}

func init() {
	r, err := route.NewStaticRoute(
		mempoolTransactionRemovedTableName,
		mempoolTransactionRemovedEventNames,
		func() route.ColumnarBatch { return newmempoolTransactionRemovedBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// mempoolTransactionRemovedFees holds the parsed wei amounts, so a bad value rejects the event before any column
// is appended.
type mempoolTransactionRemovedFees struct {
	tipCap, feeCap, blobFeeCap, replacementTipCap, replacementFeeCap proto.UInt128
}

func (b *mempoolTransactionRemovedBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetMempoolTransactionRemoved()
	if payload == nil {
		return fmt.Errorf("nil mempool_transaction_removed payload: %w", route.ErrInvalidEvent)
	}

	if payload.GetHash() == "" || payload.GetFrom() == "" || payload.GetReason() == "" {
		return fmt.Errorf("mempool_transaction_removed without hash, from or reason: %w", route.ErrInvalidEvent)
	}

	fees, err := parseMempoolTransactionRemovedFees(payload)
	if err != nil {
		return fmt.Errorf("%w: %w", err, route.ErrInvalidEvent)
	}

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.FirstSeenDateTime.Append(payload.GetFirstSeen().AsTime())
	b.LastSeenDateTime.Append(payload.GetLastSeen().AsTime())
	b.Hash.Append([]byte(payload.GetHash()))
	b.From.Append([]byte(payload.GetFrom()))

	if to := payload.GetTo(); to != "" {
		b.To.Append(proto.NewNullable([]byte(to)))
	} else {
		b.To.Append(proto.Nullable[[]byte]{})
	}

	b.Nonce.Append(payload.GetNonce().GetValue())
	b.Type.Append(uint8(payload.GetType().GetValue())) //nolint:gosec // transaction types fit in a byte
	b.Gas.Append(payload.GetGas().GetValue())
	b.GasTipCap.Append(fees.tipCap)
	b.GasFeeCap.Append(fees.feeCap)
	b.BlobGasFeeCap.Append(nullableUInt128(payload.GetBlobGasFeeCap(), fees.blobFeeCap))
	b.Reason.Append(payload.GetReason())
	b.Pool.Append(payload.GetPool())

	if hash := payload.GetReplacementHash(); hash != "" {
		b.ReplacementHash.Append(proto.NewNullable([]byte(hash)))
	} else {
		b.ReplacementHash.Append(proto.Nullable[[]byte]{})
	}

	b.ReplacementGasTipCap.Append(nullableUInt128(payload.GetReplacementGasTipCap(), fees.replacementTipCap))
	b.ReplacementGasFeeCap.Append(nullableUInt128(payload.GetReplacementGasFeeCap(), fees.replacementFeeCap))

	if nonce := payload.GetAccountNonce(); nonce != nil {
		b.AccountNonce.Append(proto.NewNullable(nonce.GetValue()))
	} else {
		b.AccountNonce.Append(proto.Nullable[uint64]{})
	}

	b.appendMetadata(event)
	b.rows++

	return nil
}

func parseMempoolTransactionRemovedFees(payload *xatu.MempoolTransactionRemoved) (mempoolTransactionRemovedFees, error) {
	var (
		fees   mempoolTransactionRemovedFees
		fields = []struct {
			name  string
			value string
			dst   *proto.UInt128
		}{
			{"gas_tip_cap", payload.GetGasTipCap(), &fees.tipCap},
			{"gas_fee_cap", payload.GetGasFeeCap(), &fees.feeCap},
			{"blob_gas_fee_cap", payload.GetBlobGasFeeCap(), &fees.blobFeeCap},
			{"replacement_gas_tip_cap", payload.GetReplacementGasTipCap(), &fees.replacementTipCap},
			{"replacement_gas_fee_cap", payload.GetReplacementGasFeeCap(), &fees.replacementFeeCap},
		}
	)

	for _, f := range fields {
		v, err := route.ParseUInt128(f.value)
		if err != nil {
			return fees, fmt.Errorf("parsing %s: %w", f.name, err)
		}

		*f.dst = v
	}

	return fees, nil
}

// nullableUInt128 is NULL when the source string was empty.
func nullableUInt128(raw string, v proto.UInt128) proto.Nullable[proto.UInt128] {
	if raw == "" {
		return proto.Nullable[proto.UInt128]{}
	}

	return proto.NewNullable(v)
}
