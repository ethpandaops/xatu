package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionLogsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_LOGS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionLogsTableName,
		canonicalExecutionLogsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionLogsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of logs) into one row per log
// in canonical_execution_logs.
func (b *canonicalExecutionLogsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalLogs()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_logs payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, item := range payload.GetLogs() {
		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(item.GetBlockNumber())
		b.TransactionIndex.Append(item.GetTransactionIndex())
		b.TransactionHash.Append([]byte(item.GetTransactionHash()))
		b.InternalIndex.Append(item.GetInternalIndex())
		b.LogIndex.Append(item.GetLogIndex())
		b.Address.Append(item.GetAddress())
		b.Topic0.Append(item.GetTopic0())
		b.Topic1.Append(ceNullStr(item.GetTopic1()))
		b.Topic2.Append(ceNullStr(item.GetTopic2()))
		b.Topic3.Append(ceNullStr(item.GetTopic3()))
		b.Data.Append(ceNullStr(item.GetData()))
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
