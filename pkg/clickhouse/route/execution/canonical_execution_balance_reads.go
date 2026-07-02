package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionBalanceReadsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_BALANCE_READS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionBalanceReadsTableName,
		canonicalExecutionBalanceReadsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionBalanceReadsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of balance reads) into one row
// per balance read in canonical_execution_balance_reads.
func (b *canonicalExecutionBalanceReadsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalBalanceReads()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_balance_reads payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, item := range payload.GetBalanceReads() {
		balance, err := route.ParseUInt256(item.GetBalance())
		if err != nil {
			return fmt.Errorf("balance read %s: %w: %w", item.GetTransactionHash(), route.ErrInvalidEvent, err)
		}

		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(item.GetBlockNumber())
		b.TransactionIndex.Append(item.GetTransactionIndex())
		b.TransactionHash.Append([]byte(item.GetTransactionHash()))
		b.InternalIndex.Append(item.GetInternalIndex())
		b.Address.Append(item.GetAddress())
		b.Balance.Append(balance)
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
