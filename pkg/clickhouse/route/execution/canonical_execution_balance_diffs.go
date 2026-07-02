package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionBalanceDiffsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_BALANCE_DIFFS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionBalanceDiffsTableName,
		canonicalExecutionBalanceDiffsEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionBalanceDiffsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of balance diffs) into one row
// per balance diff in canonical_execution_balance_diffs.
func (b *canonicalExecutionBalanceDiffsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalBalanceDiffs()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_balance_diffs payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, item := range payload.GetBalanceDiffs() {
		fromValue, err := route.ParseUInt256(item.GetFromValue())
		if err != nil {
			return fmt.Errorf("balance diff %s: %w: %w", item.GetTransactionHash(), route.ErrInvalidEvent, err)
		}

		toValue, err := route.ParseUInt256(item.GetToValue())
		if err != nil {
			return fmt.Errorf("balance diff %s: %w: %w", item.GetTransactionHash(), route.ErrInvalidEvent, err)
		}

		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(item.GetBlockNumber())
		b.TransactionIndex.Append(item.GetTransactionIndex())
		b.TransactionHash.Append([]byte(item.GetTransactionHash()))
		b.InternalIndex.Append(item.GetInternalIndex())
		b.Address.Append(item.GetAddress())
		b.FromValue.Append(fromValue)
		b.ToValue.Append(toValue)
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
