package execution

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		executionEngineNewPayloadTableName,
		[]xatu.Event_Name{xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD},
		func() flattener.ColumnarBatch {
			return newexecutionEngineNewPayloadBatch()
		},
	))
}

func (b *executionEngineNewPayloadBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionEngineNewPayload()
	if payload == nil {
		return fmt.Errorf("nil ExecutionEngineNewPayload payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.RequestedDateTime.Append(payload.GetRequestedAt().AsTime())
	b.DurationMs.Append(uint32(payload.GetDurationMs().GetValue())) //nolint:gosec // G115: duration ms fits uint32.
	b.Source.Append(payload.GetSource().String())
	b.BlockNumber.Append(payload.GetBlockNumber().GetValue())
	b.BlockHash.Append([]byte(payload.GetBlockHash()))
	b.ParentHash.Append([]byte(payload.GetParentHash()))
	b.GasUsed.Append(payload.GetGasUsed().GetValue())
	b.GasLimit.Append(payload.GetGasLimit().GetValue())
	b.TxCount.Append(payload.GetTxCount().GetValue())
	b.BlobCount.Append(payload.GetBlobCount().GetValue())
	b.Status.Append(payload.GetStatus())

	if v := payload.GetLatestValidHash(); v != "" {
		b.LatestValidHash.Append(proto.NewNullable[[]byte]([]byte(v)))
	} else {
		b.LatestValidHash.Append(proto.Nullable[[]byte]{})
	}

	if v := payload.GetValidationError(); v != "" {
		b.ValidationError.Append(proto.NewNullable[string](v))
	} else {
		b.ValidationError.Append(proto.Nullable[string]{})
	}

	b.MethodVersion.Append(payload.GetMethodVersion())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *executionEngineNewPayloadBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetExecutionEngineNewPayload()

	if payload.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetDurationMs() == nil {
		return fmt.Errorf("nil DurationMs: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
