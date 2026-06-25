package execution

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var consensusEngineApiNewPayloadEventNames = []xatu.Event_Name{
	xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD,
}

func init() {
	r, err := route.NewStaticRoute(
		consensusEngineApiNewPayloadTableName,
		consensusEngineApiNewPayloadEventNames,
		func() route.ColumnarBatch { return newconsensusEngineApiNewPayloadBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *consensusEngineApiNewPayloadBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetConsensusEngineApiNewPayload() == nil {
		return fmt.Errorf("nil consensus_engine_api_new_payload payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *consensusEngineApiNewPayloadBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetConsensusEngineApiNewPayload()

	if payload.GetDurationMs() == nil {
		return fmt.Errorf("nil DurationMs: %w", route.ErrInvalidEvent)
	}

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	if payload.GetProposerIndex() == nil {
		return fmt.Errorf("nil ProposerIndex: %w", route.ErrInvalidEvent)
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

func (b *consensusEngineApiNewPayloadBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *consensusEngineApiNewPayloadBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetConsensusEngineApiNewPayload()
	if requestedAt := payload.GetRequestedAt(); requestedAt != nil {
		b.RequestedDateTime.Append(requestedAt.AsTime().UTC())
	} else {
		b.RequestedDateTime.Append(time.Time{})
	}

	b.DurationMs.Append(payload.GetDurationMs().GetValue())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // slot values fit uint32
	b.BlockRoot.Append([]byte(payload.GetBlockRoot()))
	b.ParentBlockRoot.Append([]byte(payload.GetParentBlockRoot()))
	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue())) //nolint:gosec // proposer index fits uint32
	b.BlockNumber.Append(payload.GetBlockNumber().GetValue())
	b.BlockHash.Append([]byte(payload.GetBlockHash()))
	b.ParentHash.Append([]byte(payload.GetParentHash()))
	b.GasUsed.Append(payload.GetGasUsed().GetValue())
	b.GasLimit.Append(payload.GetGasLimit().GetValue())
	b.TxCount.Append(payload.GetTxCount().GetValue())
	b.BlobCount.Append(payload.GetBlobCount().GetValue())
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

func (b *consensusEngineApiNewPayloadBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := client.GetConsensusEngineApiNewPayload()

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
