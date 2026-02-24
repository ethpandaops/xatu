package execution

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
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

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

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

	if proposerIndex := payload.GetProposerIndex(); proposerIndex != nil {
		b.ProposerIndex.Append(uint32(proposerIndex.GetValue())) //nolint:gosec // proposer index fits uint32
	} else {
		b.ProposerIndex.Append(0)
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
