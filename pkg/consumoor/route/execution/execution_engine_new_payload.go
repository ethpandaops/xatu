package execution

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var executionEngineNewPayloadEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD,
}

func init() {
	r, err := route.NewStaticRoute(
		executionEngineNewPayloadTableName,
		executionEngineNewPayloadEventNames,
		func() route.ColumnarBatch { return newexecutionEngineNewPayloadBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *executionEngineNewPayloadBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetExecutionEngineNewPayload() == nil {
		return fmt.Errorf("nil execution_engine_new_payload payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *executionEngineNewPayloadBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetExecutionEngineNewPayload()

	if payload.GetDurationMs() == nil {
		return fmt.Errorf("nil DurationMs: %w", route.ErrInvalidEvent)
	}

	if payload.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", route.ErrInvalidEvent)
	}

	if payload.GetGasUsed() == nil {
		return fmt.Errorf("nil GasUsed: %w", route.ErrInvalidEvent)
	}

	if payload.GetGasLimit() == nil {
		return fmt.Errorf("nil GasLimit: %w", route.ErrInvalidEvent)
	}

	if payload.GetTxCount() == nil {
		return fmt.Errorf("nil TxCount: %w", route.ErrInvalidEvent)
	}

	if payload.GetBlobCount() == nil {
		return fmt.Errorf("nil BlobCount: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *executionEngineNewPayloadBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *executionEngineNewPayloadBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetExecutionEngineNewPayload()
	b.Source.Append(payload.GetSource().String())

	if requestedAt := payload.GetRequestedAt(); requestedAt != nil {
		b.RequestedDateTime.Append(requestedAt.AsTime().UTC())
	} else {
		b.RequestedDateTime.Append(time.Time{})
	}

	if durationMs := payload.GetDurationMs(); durationMs != nil {
		b.DurationMs.Append(uint32(durationMs.GetValue())) //nolint:gosec // duration fits uint32
	} else {
		b.DurationMs.Append(0)
	}

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.BlockNumber.Append(blockNumber.GetValue())
	} else {
		b.BlockNumber.Append(0)
	}

	b.BlockHash.Append([]byte(payload.GetBlockHash()))
	b.ParentHash.Append([]byte(payload.GetParentHash()))

	if gasUsed := payload.GetGasUsed(); gasUsed != nil {
		b.GasUsed.Append(gasUsed.GetValue())
	} else {
		b.GasUsed.Append(0)
	}

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.GasLimit.Append(gasLimit.GetValue())
	} else {
		b.GasLimit.Append(0)
	}

	if txCount := payload.GetTxCount(); txCount != nil {
		b.TxCount.Append(txCount.GetValue())
	} else {
		b.TxCount.Append(0)
	}

	if blobCount := payload.GetBlobCount(); blobCount != nil {
		b.BlobCount.Append(blobCount.GetValue())
	} else {
		b.BlobCount.Append(0)
	}

	b.Status.Append(payload.GetStatus())

	if lvh := payload.GetLatestValidHash(); lvh != "" {
		b.LatestValidHash.Append(proto.NewNullable[[]byte]([]byte(lvh)))
	} else {
		b.LatestValidHash.Append(proto.Nullable[[]byte]{})
	}

	if ve := payload.GetValidationError(); ve != "" {
		b.ValidationError.Append(proto.NewNullable[string](ve))
	} else {
		b.ValidationError.Append(proto.Nullable[string]{})
	}

	b.MethodVersion.Append(payload.GetMethodVersion())
}
