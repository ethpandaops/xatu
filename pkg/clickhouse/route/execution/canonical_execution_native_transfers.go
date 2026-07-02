package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionNativeTransfersEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_NATIVE_TRANSFERS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionNativeTransfersTableName,
		canonicalExecutionNativeTransfersEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionNativeTransfersBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of native transfers) into one
// row per transfer in canonical_execution_native_transfers.
func (b *canonicalExecutionNativeTransfersBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalNativeTransfers()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_native_transfers payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, item := range payload.GetNativeTransfers() {
		value, err := route.ParseUInt256(item.GetValue())
		if err != nil {
			return fmt.Errorf("native transfer %s: %w: %w", item.GetTransactionHash(), route.ErrInvalidEvent, err)
		}

		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(item.GetBlockNumber())
		b.TransactionIndex.Append(item.GetTransactionIndex())
		b.TransactionHash.Append([]byte(item.GetTransactionHash()))
		b.InternalIndex.Append(item.GetInternalIndex())
		b.TransferIndex.Append(item.GetTransferIndex())
		b.FromAddress.Append(item.GetFromAddress())
		b.ToAddress.Append(item.GetToAddress())
		b.Value.Append(value)
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
