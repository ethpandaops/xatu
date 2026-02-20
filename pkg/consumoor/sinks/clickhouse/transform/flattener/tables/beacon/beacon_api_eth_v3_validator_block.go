package beacon

import (
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV3ValidatorBlockEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV3ValidatorBlockTableName,
		beaconApiEthV3ValidatorBlockEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV3ValidatorBlockBatch() },
	))
}

func (b *beaconApiEthV3ValidatorBlockBatch) FlattenTo(
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

func (b *beaconApiEthV3ValidatorBlockBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV3ValidatorBlockBatch) appendPayload(event *xatu.DecoratedEvent) {
	eventBlock := event.GetEthV3ValidatorBlock()
	if eventBlock == nil {
		b.Slot.Append(0)
		b.BlockVersion.Append("")
		b.ExecutionPayloadBlockNumber.Append(0)
		b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
		b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})

		return
	}

	b.appendPayloadFromEventBlockV2(eventBlock)

	version := eventBlock.GetVersion()
	if version != ethv2.BlockVersion_UNKNOWN {
		b.BlockVersion.Append(strings.ToLower(version.String()))
	} else {
		b.BlockVersion.Append("")
	}
}

func (b *beaconApiEthV3ValidatorBlockBatch) appendPayloadFromEventBlockV2(
	eventBlock *ethv2.EventBlockV2,
) {
	if phase0Block := eventBlock.GetPhase0Block(); phase0Block != nil {
		if slot := phase0Block.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.appendNoExecutionPayload()

		return
	}

	if altairBlock := eventBlock.GetAltairBlock(); altairBlock != nil {
		if slot := altairBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.appendNoExecutionPayload()

		return
	}

	if bellatrixBlock := eventBlock.GetBellatrixBlock(); bellatrixBlock != nil {
		if slot := bellatrixBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.appendExecutionPayloadV2(bellatrixBlock.GetBody().GetExecutionPayload())

		return
	}

	if capellaBlock := eventBlock.GetCapellaBlock(); capellaBlock != nil {
		if slot := capellaBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.appendExecutionPayloadCapellaV2(capellaBlock.GetBody().GetExecutionPayload())

		return
	}

	if denebBlock := eventBlock.GetDenebBlock(); denebBlock != nil {
		if slot := denebBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.appendExecutionPayloadDeneb(denebBlock.GetBody().GetExecutionPayload())

		return
	}

	if electraBlock := eventBlock.GetElectraBlock(); electraBlock != nil {
		if slot := electraBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.appendExecutionPayloadElectra(electraBlock.GetBody().GetExecutionPayload())

		return
	}

	if fuluBlock := eventBlock.GetFuluBlock(); fuluBlock != nil {
		if slot := fuluBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.appendExecutionPayloadElectra(fuluBlock.GetBody().GetExecutionPayload())

		return
	}

	// Unknown block type.
	b.Slot.Append(0)
	b.appendNoExecutionPayload()
}

func (b *beaconApiEthV3ValidatorBlockBatch) appendNoExecutionPayload() {
	b.ExecutionPayloadBlockNumber.Append(0)
	b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})
}

func (b *beaconApiEthV3ValidatorBlockBatch) appendExecutionPayloadV2(
	payload *ethv1.ExecutionPayloadV2,
) {
	if payload == nil {
		b.appendNoExecutionPayload()

		return
	}

	if baseFee := payload.GetBaseFeePerGas(); baseFee != "" {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(baseFee)))
	} else {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	}

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.ExecutionPayloadBlockNumber.Append(uint32(blockNumber.GetValue())) //nolint:gosec // block number fits uint32
	} else {
		b.ExecutionPayloadBlockNumber.Append(0)
	}

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.ExecutionPayloadGasLimit.Append(proto.NewNullable[uint64](gasLimit.GetValue()))
	} else {
		b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
	}

	if gasUsed := payload.GetGasUsed(); gasUsed != nil {
		b.ExecutionPayloadGasUsed.Append(proto.NewNullable[uint64](gasUsed.GetValue()))
	} else {
		b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})
	}

	b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
}

func (b *beaconApiEthV3ValidatorBlockBatch) appendExecutionPayloadCapellaV2(
	payload *ethv1.ExecutionPayloadCapellaV2,
) {
	if payload == nil {
		b.appendNoExecutionPayload()

		return
	}

	if baseFee := payload.GetBaseFeePerGas(); baseFee != "" {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(baseFee)))
	} else {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	}

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.ExecutionPayloadBlockNumber.Append(uint32(blockNumber.GetValue())) //nolint:gosec // block number fits uint32
	} else {
		b.ExecutionPayloadBlockNumber.Append(0)
	}

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.ExecutionPayloadGasLimit.Append(proto.NewNullable[uint64](gasLimit.GetValue()))
	} else {
		b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
	}

	if gasUsed := payload.GetGasUsed(); gasUsed != nil {
		b.ExecutionPayloadGasUsed.Append(proto.NewNullable[uint64](gasUsed.GetValue()))
	} else {
		b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})
	}

	b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
}

func (b *beaconApiEthV3ValidatorBlockBatch) appendExecutionPayloadDeneb(
	payload *ethv1.ExecutionPayloadDeneb,
) {
	if payload == nil {
		b.appendNoExecutionPayload()

		return
	}

	if baseFee := payload.GetBaseFeePerGas(); baseFee != "" {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(baseFee)))
	} else {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	}

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.ExecutionPayloadBlockNumber.Append(uint32(blockNumber.GetValue())) //nolint:gosec // block number fits uint32
	} else {
		b.ExecutionPayloadBlockNumber.Append(0)
	}

	if blobGasUsed := payload.GetBlobGasUsed(); blobGasUsed != nil {
		b.ExecutionPayloadBlobGasUsed.Append(proto.NewNullable[uint64](blobGasUsed.GetValue()))
	} else {
		b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
	}

	if excessBlobGas := payload.GetExcessBlobGas(); excessBlobGas != nil {
		b.ExecutionPayloadExcessBlobGas.Append(proto.NewNullable[uint64](excessBlobGas.GetValue()))
	} else {
		b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
	}

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.ExecutionPayloadGasLimit.Append(proto.NewNullable[uint64](gasLimit.GetValue()))
	} else {
		b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
	}

	if gasUsed := payload.GetGasUsed(); gasUsed != nil {
		b.ExecutionPayloadGasUsed.Append(proto.NewNullable[uint64](gasUsed.GetValue()))
	} else {
		b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})
	}
}

func (b *beaconApiEthV3ValidatorBlockBatch) appendExecutionPayloadElectra(
	payload *ethv1.ExecutionPayloadElectra,
) {
	if payload == nil {
		b.appendNoExecutionPayload()

		return
	}

	if baseFee := payload.GetBaseFeePerGas(); baseFee != "" {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(baseFee)))
	} else {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	}

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.ExecutionPayloadBlockNumber.Append(uint32(blockNumber.GetValue())) //nolint:gosec // block number fits uint32
	} else {
		b.ExecutionPayloadBlockNumber.Append(0)
	}

	if blobGasUsed := payload.GetBlobGasUsed(); blobGasUsed != nil {
		b.ExecutionPayloadBlobGasUsed.Append(proto.NewNullable[uint64](blobGasUsed.GetValue()))
	} else {
		b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
	}

	if excessBlobGas := payload.GetExcessBlobGas(); excessBlobGas != nil {
		b.ExecutionPayloadExcessBlobGas.Append(proto.NewNullable[uint64](excessBlobGas.GetValue()))
	} else {
		b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
	}

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.ExecutionPayloadGasLimit.Append(proto.NewNullable[uint64](gasLimit.GetValue()))
	} else {
		b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
	}

	if gasUsed := payload.GetGasUsed(); gasUsed != nil {
		b.ExecutionPayloadGasUsed.Append(proto.NewNullable[uint64](gasUsed.GetValue()))
	} else {
		b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})
	}
}

func (b *beaconApiEthV3ValidatorBlockBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Epoch.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
		b.ConsensusPayloadValue.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadValue.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})

		return
	}

	additional := event.GetMeta().GetClient().GetEthV3ValidatorBlock()
	if additional == nil {
		b.Epoch.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
		b.ConsensusPayloadValue.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadValue.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})

		return
	}

	if epochNumber := additional.GetEpoch().GetNumber(); epochNumber != nil {
		b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // epoch fits uint32
	} else {
		b.Epoch.Append(0)
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
		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.EpochStartDateTime.Append(time.Time{})
	}

	if totalBytes := additional.GetTotalBytes(); totalBytes != nil {
		b.BlockTotalBytes.Append(proto.NewNullable[uint32](uint32(totalBytes.GetValue()))) //nolint:gosec // byte count fits uint32
	} else {
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if totalBytesCompressed := additional.GetTotalBytesCompressed(); totalBytesCompressed != nil {
		b.BlockTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(totalBytesCompressed.GetValue()))) //nolint:gosec // byte count fits uint32
	} else {
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}

	if v, err := strconv.ParseUint(additional.GetConsensusValue(), 10, 64); err == nil {
		b.ConsensusPayloadValue.Append(proto.NewNullable[uint64](v))
	} else {
		b.ConsensusPayloadValue.Append(proto.Nullable[uint64]{})
	}

	if v, err := strconv.ParseUint(additional.GetExecutionValue(), 10, 64); err == nil {
		b.ExecutionPayloadValue.Append(proto.NewNullable[uint64](v))
	} else {
		b.ExecutionPayloadValue.Append(proto.Nullable[uint64]{})
	}

	if transactionsCount := additional.GetTransactionsCount(); transactionsCount != nil {
		b.ExecutionPayloadTransactionsCount.Append(proto.NewNullable[uint32](uint32(transactionsCount.GetValue()))) //nolint:gosec // tx count fits uint32
	} else {
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
	}

	if transactionsTotalBytes := additional.GetTransactionsTotalBytes(); transactionsTotalBytes != nil {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.NewNullable[uint32](uint32(transactionsTotalBytes.GetValue()))) //nolint:gosec // byte count fits uint32
	} else {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if v := additional.GetTransactionsTotalBytesCompressed(); v != nil {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(v.GetValue()))) //nolint:gosec // byte count fits uint32
	} else {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}
}
