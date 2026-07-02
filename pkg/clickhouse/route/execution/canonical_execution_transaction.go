package execution

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionTransactionEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_TRANSACTION,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionTransactionTableName,
		canonicalExecutionTransactionEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionTransactionBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of transactions) into one row
// per transaction in canonical_execution_transaction.
func (b *canonicalExecutionTransactionBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalTransaction()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_transaction payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, tx := range payload.GetTransactions() {
		value, err := route.ParseUInt256(tx.GetValue())
		if err != nil {
			return fmt.Errorf("transaction %s: %w: %w", tx.GetTransactionHash(), route.ErrInvalidEvent, err)
		}

		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(tx.GetBlockNumber())
		b.TransactionIndex.Append(tx.GetTransactionIndex())
		b.TransactionHash.Append([]byte(tx.GetTransactionHash()))
		b.Nonce.Append(tx.GetNonce())
		b.FromAddress.Append(tx.GetFromAddress())
		b.ToAddress.Append(ceNullStr(tx.GetToAddress()))
		b.Value.Append(value)
		b.InputData.Append(ceNullStr(tx.GetInput()))
		b.GasLimit.Append(tx.GetGasLimit())
		b.GasUsed.Append(tx.GetGasUsed())
		b.GasPrice.Append(proto.UInt128{Low: tx.GetGasPrice()})
		b.TransactionType.Append(uint8(tx.GetTransactionType())) //nolint:gosec // transaction type is 0-4, fits uint8.
		b.MaxPriorityFeePerGas.Append(tx.GetMaxPriorityFeePerGas())
		b.MaxFeePerGas.Append(tx.GetMaxFeePerGas())
		b.Success.Append(tx.GetSuccess())
		b.NInputBytes.Append(tx.GetNInputBytes())
		b.NInputZeroBytes.Append(tx.GetNInputZeroBytes())
		b.NInputNonzeroBytes.Append(tx.GetNInputNonzeroBytes())
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
