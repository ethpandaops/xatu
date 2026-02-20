package execution

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var executionEngineGetBlobsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_ENGINE_GET_BLOBS,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		executionEngineGetBlobsTableName,
		executionEngineGetBlobsEventNames,
		func() flattener.ColumnarBatch { return newexecutionEngineGetBlobsBatch() },
	))
}

func (b *executionEngineGetBlobsBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event)
	b.rows++

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
	if payload == nil {
		b.RequestedDateTime.Append(time.Time{})
		b.DurationMs.Append(0)
		b.Source.Append("")
		b.RequestedCount.Append(0)
		b.VersionedHashes.Append(nil)
		b.ReturnedCount.Append(0)
		b.ReturnedBlobIndexes.Append(nil)
		b.Status.Append("")
		b.ErrorMessage.Append(proto.Nullable[string]{})
		b.MethodVersion.Append("")

		return
	}

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
