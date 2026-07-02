package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionFourByteCountsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_FOUR_BYTE_COUNTS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionFourByteCountsTableName,
		canonicalExecutionFourByteCountsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionFourByteCountsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of four_byte_counts) into one
// row per four_byte_count in canonical_execution_four_byte_counts.
func (b *canonicalExecutionFourByteCountsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalFourByteCounts()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_four_byte_counts payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, item := range payload.GetFourByteCounts() {
		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(item.GetBlockNumber())
		b.TransactionIndex.Append(item.GetTransactionIndex())
		b.TransactionHash.Append([]byte(item.GetTransactionHash()))
		b.Signature.Append(item.GetSignature())
		b.Size.Append(item.GetSize())
		b.Count.Append(item.GetCount())
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
