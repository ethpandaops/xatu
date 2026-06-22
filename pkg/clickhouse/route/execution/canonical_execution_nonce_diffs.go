package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionNonceDiffsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_NONCE_DIFFS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionNonceDiffsTableName,
		canonicalExecutionNonceDiffsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionNonceDiffsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of nonce diffs) into one row
// per nonce diff in canonical_execution_nonce_diffs.
func (b *canonicalExecutionNonceDiffsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalNonceDiffs()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_nonce_diffs payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, diff := range payload.GetNonceDiffs() {
		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(diff.GetBlockNumber())
		b.TransactionIndex.Append(diff.GetTransactionIndex())
		b.TransactionHash.Append([]byte(diff.GetTransactionHash()))
		b.InternalIndex.Append(diff.GetInternalIndex())
		b.Address.Append(diff.GetAddress())
		b.FromValue.Append(diff.GetFromValue())
		b.ToValue.Append(diff.GetToValue())
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
