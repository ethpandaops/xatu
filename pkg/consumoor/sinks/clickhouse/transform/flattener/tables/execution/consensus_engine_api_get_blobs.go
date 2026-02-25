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
		consensusEngineApiGetBlobsTableName,
		[]xatu.Event_Name{xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS},
		func() flattener.ColumnarBatch {
			return newconsensusEngineApiGetBlobsBatch()
		},
	))
}

func (b *consensusEngineApiGetBlobsBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetConsensusEngineApiGetBlobs()
	if payload == nil {
		return fmt.Errorf("nil ConsensusEngineApiGetBlobs payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetConsensusEngineApiGetBlobs()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.RequestedDateTime.Append(payload.GetRequestedAt().AsTime())
	b.DurationMs.Append(payload.GetDurationMs().GetValue())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(payload.GetBlockRoot()))
	b.ParentBlockRoot.Append([]byte(payload.GetParentBlockRoot()))
	b.RequestedCount.Append(payload.GetRequestedCount().GetValue())
	b.VersionedHashes.Append(flattener.StringsToBytes(payload.GetVersionedHashes()))
	b.ReturnedCount.Append(payload.GetReturnedCount().GetValue())
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

func (b *consensusEngineApiGetBlobsBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetConsensusEngineApiGetBlobs()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetDurationMs() == nil {
		return fmt.Errorf("nil DurationMs: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetConsensusEngineApiGetBlobs()
	if addl == nil {
		return fmt.Errorf("nil ConsensusEngineApiGetBlobs additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
