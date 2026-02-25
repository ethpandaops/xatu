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
		executionEngineGetBlobsTableName,
		[]xatu.Event_Name{xatu.Event_EXECUTION_ENGINE_GET_BLOBS},
		func() flattener.ColumnarBatch {
			return newexecutionEngineGetBlobsBatch()
		},
	))
}

func (b *executionEngineGetBlobsBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionEngineGetBlobs()
	if payload == nil {
		return fmt.Errorf("nil ExecutionEngineGetBlobs payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.RequestedDateTime.Append(payload.GetRequestedAt().AsTime())
	b.DurationMs.Append(uint32(payload.GetDurationMs().GetValue())) //nolint:gosec // G115: duration ms fits uint32.
	b.Source.Append(payload.GetSource().String())
	b.RequestedCount.Append(payload.GetRequestedCount().GetValue())
	b.VersionedHashes.Append(flattener.StringsToBytes(payload.GetVersionedHashes()))
	b.ReturnedCount.Append(payload.GetReturnedCount().GetValue())

	// Convert []*wrapperspb.UInt32Value to []uint8 for the array column.
	indexes := payload.GetReturnedBlobIndexes()

	idxSlice := make([]uint8, 0, len(indexes))
	for _, idx := range indexes {
		idxSlice = append(idxSlice, uint8(idx.GetValue())) //nolint:gosec // G115: blob index fits uint8.
	}

	b.ReturnedBlobIndexes.Append(idxSlice)

	b.Status.Append(payload.GetStatus())

	if v := payload.GetErrorMessage(); v != "" {
		b.ErrorMessage.Append(proto.NewNullable[string](v))
	} else {
		b.ErrorMessage.Append(proto.Nullable[string]{})
	}

	b.MethodVersion.Append(payload.GetMethodVersion())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *executionEngineGetBlobsBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetExecutionEngineGetBlobs()

	if payload.GetDurationMs() == nil {
		return fmt.Errorf("nil DurationMs: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
