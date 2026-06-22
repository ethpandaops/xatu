package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionErc20TransfersEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_ERC20_TRANSFERS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionErc20TransfersTableName,
		canonicalExecutionErc20TransfersEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionErc20TransfersBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of erc20 transfers) into one
// row per transfer in canonical_execution_erc20_transfers.
func (b *canonicalExecutionErc20TransfersBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalErc20Transfers()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_erc20_transfers payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, transfer := range payload.GetErc20Transfers() {
		value, err := route.ParseUInt256(transfer.GetValue())
		if err != nil {
			return fmt.Errorf("erc20_transfer %s: %w: %w", transfer.GetTransactionHash(), route.ErrInvalidEvent, err)
		}

		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(transfer.GetBlockNumber())
		b.TransactionIndex.Append(transfer.GetTransactionIndex())
		b.TransactionHash.Append([]byte(transfer.GetTransactionHash()))
		b.InternalIndex.Append(transfer.GetInternalIndex())
		b.LogIndex.Append(transfer.GetLogIndex())
		b.Erc20.Append(transfer.GetErc20())
		b.FromAddress.Append(transfer.GetFromAddress())
		b.ToAddress.Append(transfer.GetToAddress())
		b.Value.Append(value)
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
