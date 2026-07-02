package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionErc721TransfersEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_ERC721_TRANSFERS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionErc721TransfersTableName,
		canonicalExecutionErc721TransfersEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionErc721TransfersBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of erc721 transfers) into one
// row per transfer in canonical_execution_erc721_transfers.
func (b *canonicalExecutionErc721TransfersBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalErc721Transfers()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_erc721_transfers payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, transfer := range payload.GetErc721Transfers() {
		token, err := route.ParseUInt256(transfer.GetToken())
		if err != nil {
			return fmt.Errorf("erc721 transfer %s: %w: %w", transfer.GetTransactionHash(), route.ErrInvalidEvent, err)
		}

		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(transfer.GetBlockNumber())
		b.TransactionIndex.Append(transfer.GetTransactionIndex())
		b.TransactionHash.Append([]byte(transfer.GetTransactionHash()))
		b.InternalIndex.Append(transfer.GetInternalIndex())
		b.LogIndex.Append(transfer.GetLogIndex())
		b.Erc721.Append(transfer.GetErc721())
		b.FromAddress.Append(transfer.GetFromAddress())
		b.ToAddress.Append(transfer.GetToAddress())
		b.Token.Append(token)
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
