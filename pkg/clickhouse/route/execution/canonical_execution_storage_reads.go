package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionStorageReadsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_STORAGE_READS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionStorageReadsTableName,
		canonicalExecutionStorageReadsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionStorageReadsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of storage reads) into one row
// per storage read in canonical_execution_storage_reads.
func (b *canonicalExecutionStorageReadsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalStorageReads()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_storage_reads payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, item := range payload.GetStorageReads() {
		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(item.GetBlockNumber())
		b.TransactionIndex.Append(item.GetTransactionIndex())
		b.TransactionHash.Append([]byte(item.GetTransactionHash()))
		b.InternalIndex.Append(item.GetInternalIndex())
		b.ContractAddress.Append(item.GetContractAddress())
		b.Slot.Append(item.GetSlot())
		b.Value.Append(item.GetValue())
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
