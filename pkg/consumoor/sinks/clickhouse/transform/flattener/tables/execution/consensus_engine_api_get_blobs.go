package execution

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var consensusEngineApiGetBlobsEventNames = []xatu.Event_Name{
	xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		consensusEngineApiGetBlobsTableName,
		consensusEngineApiGetBlobsEventNames,
		func() flattener.ColumnarBatch { return newconsensusEngineApiGetBlobsBatch() },
	))
}

func (b *consensusEngineApiGetBlobsBatch) FlattenTo(
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
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *consensusEngineApiGetBlobsBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *consensusEngineApiGetBlobsBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetConsensusEngineApiGetBlobs()
	if payload == nil {
		b.RequestedDateTime.Append(time.Time{})
		b.DurationMs.Append(0)
		b.Slot.Append(0)
		b.BlockRoot.Append(nil)
		b.ParentBlockRoot.Append(nil)
		b.RequestedCount.Append(0)
		b.VersionedHashes.Append(nil)
		b.ReturnedCount.Append(0)
		b.Status.Append("")
		b.ErrorMessage.Append(proto.Nullable[string]{})
		b.MethodVersion.Append("")

		return
	}

	if requestedAt := payload.GetRequestedAt(); requestedAt != nil {
		b.RequestedDateTime.Append(requestedAt.AsTime().UTC())
	} else {
		b.RequestedDateTime.Append(time.Time{})
	}

	if durationMs := payload.GetDurationMs(); durationMs != nil {
		b.DurationMs.Append(durationMs.GetValue())
	} else {
		b.DurationMs.Append(0)
	}

	if slot := payload.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot values fit uint32
	} else {
		b.Slot.Append(0)
	}

	b.BlockRoot.Append([]byte(payload.GetBlockRoot()))
	b.ParentBlockRoot.Append([]byte(payload.GetParentBlockRoot()))

	if requestedCount := payload.GetRequestedCount(); requestedCount != nil {
		b.RequestedCount.Append(requestedCount.GetValue())
	} else {
		b.RequestedCount.Append(0)
	}

	hashes := payload.GetVersionedHashes()

	hashBytes := make([][]byte, len(hashes))
	for i, h := range hashes {
		hashBytes[i] = []byte(h)
	}

	b.VersionedHashes.Append(hashBytes)

	if returnedCount := payload.GetReturnedCount(); returnedCount != nil {
		b.ReturnedCount.Append(returnedCount.GetValue())
	} else {
		b.ReturnedCount.Append(0)
	}

	b.Status.Append(payload.GetStatus())

	if errMsg := payload.GetErrorMessage(); errMsg != "" {
		b.ErrorMessage.Append(proto.NewNullable[string](errMsg))
	} else {
		b.ErrorMessage.Append(proto.Nullable[string]{})
	}

	b.MethodVersion.Append(payload.GetMethodVersion())
}

func (b *consensusEngineApiGetBlobsBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := client.GetConsensusEngineApiGetBlobs()

	if additional == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	if slot := additional.GetSlot(); slot != nil {
		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.SlotStartDateTime.Append(time.Time{})
	}

	if epoch := additional.GetEpoch(); epoch != nil {
		if epochNumber := epoch.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // epoch values fit uint32
		} else {
			b.Epoch.Append(0)
		}

		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
	}
}
