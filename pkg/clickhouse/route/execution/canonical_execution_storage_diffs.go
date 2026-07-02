package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionStorageDiffsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_STORAGE_DIFFS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionStorageDiffsTableName,
		canonicalExecutionStorageDiffsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionStorageDiffsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of storage diffs) into one row
// per storage diff in canonical_execution_storage_diffs.
func (b *canonicalExecutionStorageDiffsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalStorageDiffs()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_storage_diffs payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, sd := range payload.GetStorageDiffs() {
		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(sd.GetBlockNumber())
		b.TransactionIndex.Append(sd.GetTransactionIndex())
		b.TransactionHash.Append([]byte(sd.GetTransactionHash()))
		b.InternalIndex.Append(sd.GetInternalIndex())
		b.Address.Append(sd.GetAddress())
		b.Slot.Append(sd.GetSlot())
		b.FromValue.Append(sd.GetFromValue())
		b.ToValue.Append(sd.GetToValue())
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
