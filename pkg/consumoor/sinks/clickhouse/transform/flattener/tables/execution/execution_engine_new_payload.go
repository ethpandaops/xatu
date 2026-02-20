package execution

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var executionEngineNewPayloadEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		executionEngineNewPayloadTableName,
		executionEngineNewPayloadEventNames,
		func() flattener.ColumnarBatch { return newexecutionEngineNewPayloadBatch() },
	))
}

func (b *executionEngineNewPayloadBatch) FlattenTo(
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
	if payload == nil {
		b.RequestedDateTime.Append(time.Time{})
		b.DurationMs.Append(0)
		b.Source.Append("")
		b.BlockNumber.Append(0)
		b.BlockHash.Append(nil)
		b.ParentHash.Append(nil)
		b.GasUsed.Append(0)
		b.GasLimit.Append(0)
		b.TxCount.Append(0)
		b.BlobCount.Append(0)
		b.Status.Append("")
		b.LatestValidHash.Append(proto.Nullable[[]byte]{})
		b.ValidationError.Append(proto.Nullable[string]{})
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
