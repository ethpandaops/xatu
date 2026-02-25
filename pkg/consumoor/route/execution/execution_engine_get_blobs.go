package execution

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var executionEngineGetBlobsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_ENGINE_GET_BLOBS,
}

func init() {
	r, err := route.NewStaticRoute(
		executionEngineGetBlobsTableName,
		executionEngineGetBlobsEventNames,
		func() route.ColumnarBatch { return newexecutionEngineGetBlobsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *executionEngineGetBlobsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetExecutionEngineGetBlobs() == nil {
		return fmt.Errorf("nil execution_engine_get_blobs payload: %w", route.ErrInvalidEvent)
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

func (b *executionEngineGetBlobsBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetExecutionEngineGetBlobs()

	if payload.GetDurationMs() == nil {
		return fmt.Errorf("nil DurationMs: %w", route.ErrInvalidEvent)
	}

	if payload.GetRequestedCount() == nil {
		return fmt.Errorf("nil RequestedCount: %w", route.ErrInvalidEvent)
	}

	if payload.GetReturnedCount() == nil {
		return fmt.Errorf("nil ReturnedCount: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *executionEngineGetBlobsBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *executionEngineGetBlobsBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetExecutionEngineGetBlobs()
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

	if requestedCount := payload.GetRequestedCount(); requestedCount != nil {
		b.RequestedCount.Append(requestedCount.GetValue())
	} else {
		b.RequestedCount.Append(0)
	}

	strs := payload.GetVersionedHashes()

	vhBytes := make([][]byte, len(strs))
	for i, s := range strs {
		vhBytes[i] = []byte(s)
	}

	b.VersionedHashes.Append(vhBytes)

	if returnedCount := payload.GetReturnedCount(); returnedCount != nil {
		b.ReturnedCount.Append(returnedCount.GetValue())
	} else {
		b.ReturnedCount.Append(0)
	}

	indexes := payload.GetReturnedBlobIndexes()
	if len(indexes) > 0 {
		blobIndexes := make([]uint8, 0, len(indexes))

		for _, idx := range indexes {
			if idx != nil {
				blobIndexes = append(blobIndexes, uint8(min(idx.GetValue(), 255))) //nolint:gosec // bounded by min
			}
		}

		b.ReturnedBlobIndexes.Append(blobIndexes)
	} else {
		b.ReturnedBlobIndexes.Append(nil)
	}

	b.Status.Append(payload.GetStatus())

	if errMsg := payload.GetErrorMessage(); errMsg != "" {
		b.ErrorMessage.Append(proto.NewNullable[string](errMsg))
	} else {
		b.ErrorMessage.Append(proto.Nullable[string]{})
	}

	b.MethodVersion.Append(payload.GetMethodVersion())
}
