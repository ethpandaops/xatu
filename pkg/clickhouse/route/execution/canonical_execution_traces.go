package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionTracesEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_TRACES,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionTracesTableName,
		canonicalExecutionTracesEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionTracesBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of traces) into one row per
// trace in canonical_execution_traces.
func (b *canonicalExecutionTracesBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalTraces()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_traces payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, item := range payload.GetTraces() {
		actionValue, err := route.ParseUInt256(item.GetActionValue())
		if err != nil {
			return fmt.Errorf("trace %s: %w: %w", item.GetTransactionHash(), route.ErrInvalidEvent, err)
		}

		b.UpdatedDateTime.Append(now)
		b.BlockNumber.Append(item.GetBlockNumber())
		b.TransactionIndex.Append(item.GetTransactionIndex())
		b.TransactionHash.Append([]byte(item.GetTransactionHash()))
		b.InternalIndex.Append(item.GetInternalIndex())
		b.ActionFrom.Append(item.GetActionFrom())
		b.ActionTo.Append(ceNullStr(item.GetActionTo()))
		b.ActionValue.Append(actionValue)
		b.ActionGas.Append(item.GetActionGas())
		b.ActionInput.Append(ceNullStr(item.GetActionInput()))
		b.ActionCallType.Append(item.GetActionCallType())
		b.ActionInit.Append(ceNullStr(item.GetActionInit()))
		b.ActionRewardType.Append(item.GetActionRewardType())
		b.ActionType.Append(item.GetActionType())
		b.ResultGasUsed.Append(item.GetResultGasUsed())
		b.ResultOutput.Append(ceNullStr(item.GetResultOutput()))
		b.ResultCode.Append(ceNullStr(item.GetResultCode()))
		b.ResultAddress.Append(ceNullStr(item.GetResultAddress()))
		b.TraceAddress.Append(ceNullStr(item.GetTraceAddress()))
		b.Subtraces.Append(item.GetSubtraces())
		b.Error.Append(ceNullStr(item.GetError()))
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}
