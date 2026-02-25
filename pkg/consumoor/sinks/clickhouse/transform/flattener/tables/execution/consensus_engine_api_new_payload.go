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
		consensusEngineApiNewPayloadTableName,
		[]xatu.Event_Name{xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD},
		func() flattener.ColumnarBatch {
			return newconsensusEngineApiNewPayloadBatch()
		},
	))
}

func (b *consensusEngineApiNewPayloadBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetConsensusEngineApiNewPayload()
	if payload == nil {
		return fmt.Errorf("nil ConsensusEngineApiNewPayload payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetConsensusEngineApiNewPayload()

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
	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue())) //nolint:gosec // G115: proposer index fits uint32.
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

func (b *consensusEngineApiNewPayloadBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetConsensusEngineApiNewPayload()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetDurationMs() == nil {
		return fmt.Errorf("nil DurationMs: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetConsensusEngineApiNewPayload()
	if addl == nil {
		return fmt.Errorf("nil ConsensusEngineApiNewPayload additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
