package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionNonceReadsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_NONCE_READS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionNonceReadsTableName,
		canonicalExecutionNonceReadsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionNonceReadsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of nonce reads) into one row
// per nonce read in canonical_execution_nonce_reads.
func (b *canonicalExecutionNonceReadsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalNonceReads()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_nonce_reads payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, nr := range payload.GetNonceReads() {
		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(nr.GetBlockNumber())
		b.TransactionIndex.Append(nr.GetTransactionIndex())
		b.TransactionHash.Append([]byte(nr.GetTransactionHash()))
		b.InternalIndex.Append(nr.GetInternalIndex())
		b.Address.Append(nr.GetAddress())
		b.Nonce.Append(nr.GetNonce())
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
