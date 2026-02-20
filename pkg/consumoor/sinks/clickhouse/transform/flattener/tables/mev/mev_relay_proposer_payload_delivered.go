package mev

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var mevRelayProposerPayloadDeliveredEventNames = []xatu.Event_Name{
	xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		mevRelayProposerPayloadDeliveredTableName,
		mevRelayProposerPayloadDeliveredEventNames,
		func() flattener.ColumnarBatch { return newmevRelayProposerPayloadDeliveredBatch() },
	))
}

func (b *mevRelayProposerPayloadDeliveredBatch) FlattenTo(
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

func (b *mevRelayProposerPayloadDeliveredBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *mevRelayProposerPayloadDeliveredBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetMevRelayPayloadDelivered()
	if payload == nil {
		b.appendZeroPayload()

		return
	}

	if slot := payload.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
	} else {
		b.Slot.Append(0)
	}

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.BlockNumber.Append(blockNumber.GetValue())
	} else {
		b.BlockNumber.Append(0)
	}

	if blockHash := payload.GetBlockHash(); blockHash != nil {
		b.BlockHash.Append([]byte(blockHash.GetValue()))
	} else {
		b.BlockHash.Append(nil)
	}

	if proposerPubkey := payload.GetProposerPubkey(); proposerPubkey != nil {
		b.ProposerPubkey.Append(proposerPubkey.GetValue())
	} else {
		b.ProposerPubkey.Append("")
	}

	if builderPubkey := payload.GetBuilderPubkey(); builderPubkey != nil {
		b.BuilderPubkey.Append(builderPubkey.GetValue())
	} else {
		b.BuilderPubkey.Append("")
	}

	if proposerFeeRecipient := payload.GetProposerFeeRecipient(); proposerFeeRecipient != nil {
		b.ProposerFeeRecipient.Append([]byte(proposerFeeRecipient.GetValue()))
	} else {
		b.ProposerFeeRecipient.Append(nil)
	}

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.GasLimit.Append(gasLimit.GetValue())
	} else {
		b.GasLimit.Append(0)
	}

	if gasUsed := payload.GetGasUsed(); gasUsed != nil {
		b.GasUsed.Append(gasUsed.GetValue())
	} else {
		b.GasUsed.Append(0)
	}

	if value := payload.GetValue(); value != nil {
		b.Value.Append(flattener.ParseUInt256(value.GetValue()))
	} else {
		b.Value.Append(proto.UInt256{})
	}

	if numTx := payload.GetNumTx(); numTx != nil {
		b.NumTx.Append(uint32(numTx.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
	} else {
		b.NumTx.Append(0)
	}
}

func (b *mevRelayProposerPayloadDeliveredBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.appendZeroAdditionalData()

		return
	}

	client := event.GetMeta().GetClient()
	additional := client.GetMevRelayPayloadDelivered()

	if additional == nil {
		b.appendZeroAdditionalData()

		return
	}

	if relay := additional.GetRelay(); relay != nil {
		if name := relay.GetName(); name != nil {
			b.RelayName.Append(name.GetValue())
		} else {
			b.RelayName.Append("")
		}
	} else {
		b.RelayName.Append("")
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
			b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
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

	if wallclockSlot := additional.GetWallclockSlot(); wallclockSlot != nil {
		if slotNumber := wallclockSlot.GetNumber(); slotNumber != nil {
			b.WallclockSlot.Append(uint32(slotNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
		} else {
			b.WallclockSlot.Append(0)
		}

		if startDateTime := wallclockSlot.GetStartDateTime(); startDateTime != nil {
			b.WallclockSlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.WallclockSlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.WallclockSlot.Append(0)
		b.WallclockSlotStartDateTime.Append(time.Time{})
	}

	if wallclockEpoch := additional.GetWallclockEpoch(); wallclockEpoch != nil {
		if epochNumber := wallclockEpoch.GetNumber(); epochNumber != nil {
			b.WallclockEpoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
		} else {
			b.WallclockEpoch.Append(0)
		}

		if startDateTime := wallclockEpoch.GetStartDateTime(); startDateTime != nil {
			b.WallclockEpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.WallclockEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.WallclockEpoch.Append(0)
		b.WallclockEpochStartDateTime.Append(time.Time{})
	}
}

func (b *mevRelayProposerPayloadDeliveredBatch) appendZeroPayload() {
	b.Slot.Append(0)
	b.BlockNumber.Append(0)
	b.BlockHash.Append(nil)
	b.ProposerPubkey.Append("")
	b.BuilderPubkey.Append("")
	b.ProposerFeeRecipient.Append(nil)
	b.GasLimit.Append(0)
	b.GasUsed.Append(0)
	b.Value.Append(proto.UInt256{})
	b.NumTx.Append(0)
}

func (b *mevRelayProposerPayloadDeliveredBatch) appendZeroAdditionalData() {
	b.RelayName.Append("")
	b.SlotStartDateTime.Append(time.Time{})
	b.Epoch.Append(0)
	b.EpochStartDateTime.Append(time.Time{})
	b.WallclockSlot.Append(0)
	b.WallclockSlotStartDateTime.Append(time.Time{})
	b.WallclockEpoch.Append(0)
	b.WallclockEpochStartDateTime.Append(time.Time{})
}
