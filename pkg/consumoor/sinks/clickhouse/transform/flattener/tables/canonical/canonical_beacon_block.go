package canonical

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
}

var canonicalBeaconBlockPredicate = func(event *xatu.DecoratedEvent) bool {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return false
	}

	return event.GetMeta().GetClient().GetEthV2BeaconBlockV2().GetFinalizedWhenRequested()
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockTableName,
		canonicalBeaconBlockEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconBlockBatch() },
		flattener.WithStaticRoutePredicate(canonicalBeaconBlockPredicate),
	))
}

func (b *canonicalBeaconBlockBatch) FlattenTo(
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

func (b *canonicalBeaconBlockBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockBatch) appendPayload(event *xatu.DecoratedEvent) {
	eventBlock := event.GetEthV2BeaconBlockV2()
	if eventBlock == nil {
		b.Slot.Append(0)
		b.ProposerIndex.Append(0)
		b.ParentRoot.Append(nil)
		b.StateRoot.Append(nil)
		b.Eth1DataBlockHash.Append(nil)
		b.Eth1DataDepositRoot.Append(nil)
		b.appendNullExecutionPayload()

		return
	}

	b.appendPayloadFromEventBlockV2(eventBlock)
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockBatch) appendPayloadFromEventBlockV2(eventBlock *ethv2.EventBlockV2) {
	if phase0Block := eventBlock.GetPhase0Block(); phase0Block != nil {
		if slot := phase0Block.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue()))
		} else {
			b.Slot.Append(0)
		}

		if proposerIndex := phase0Block.GetProposerIndex(); proposerIndex != nil {
			b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
		} else {
			b.ProposerIndex.Append(0)
		}

		b.ParentRoot.Append([]byte(phase0Block.GetParentRoot()))
		b.StateRoot.Append([]byte(phase0Block.GetStateRoot()))
		b.appendEth1Data(phase0Block.GetBody().GetEth1Data())
		b.appendNullExecutionPayload()

		return
	}

	if altairBlock := eventBlock.GetAltairBlock(); altairBlock != nil {
		if slot := altairBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue()))
		} else {
			b.Slot.Append(0)
		}

		if proposerIndex := altairBlock.GetProposerIndex(); proposerIndex != nil {
			b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
		} else {
			b.ProposerIndex.Append(0)
		}

		b.ParentRoot.Append([]byte(altairBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(altairBlock.GetStateRoot()))
		b.appendEth1Data(altairBlock.GetBody().GetEth1Data())
		b.appendNullExecutionPayload()

		return
	}

	if bellatrixBlock := eventBlock.GetBellatrixBlock(); bellatrixBlock != nil {
		if slot := bellatrixBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue()))
		} else {
			b.Slot.Append(0)
		}

		if proposerIndex := bellatrixBlock.GetProposerIndex(); proposerIndex != nil {
			b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
		} else {
			b.ProposerIndex.Append(0)
		}

		b.ParentRoot.Append([]byte(bellatrixBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(bellatrixBlock.GetStateRoot()))

		body := bellatrixBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadV2(body.GetExecutionPayload())

		return
	}

	if capellaBlock := eventBlock.GetCapellaBlock(); capellaBlock != nil {
		if slot := capellaBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue()))
		} else {
			b.Slot.Append(0)
		}

		if proposerIndex := capellaBlock.GetProposerIndex(); proposerIndex != nil {
			b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
		} else {
			b.ProposerIndex.Append(0)
		}

		b.ParentRoot.Append([]byte(capellaBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(capellaBlock.GetStateRoot()))

		body := capellaBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadCapellaV2(body.GetExecutionPayload())

		return
	}

	if denebBlock := eventBlock.GetDenebBlock(); denebBlock != nil {
		if slot := denebBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue()))
		} else {
			b.Slot.Append(0)
		}

		if proposerIndex := denebBlock.GetProposerIndex(); proposerIndex != nil {
			b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
		} else {
			b.ProposerIndex.Append(0)
		}

		b.ParentRoot.Append([]byte(denebBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(denebBlock.GetStateRoot()))

		body := denebBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadDeneb(body.GetExecutionPayload())

		return
	}

	if electraBlock := eventBlock.GetElectraBlock(); electraBlock != nil {
		if slot := electraBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue()))
		} else {
			b.Slot.Append(0)
		}

		if proposerIndex := electraBlock.GetProposerIndex(); proposerIndex != nil {
			b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
		} else {
			b.ProposerIndex.Append(0)
		}

		b.ParentRoot.Append([]byte(electraBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(electraBlock.GetStateRoot()))

		body := electraBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadElectra(body.GetExecutionPayload())

		return
	}

	if fuluBlock := eventBlock.GetFuluBlock(); fuluBlock != nil {
		if slot := fuluBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue()))
		} else {
			b.Slot.Append(0)
		}

		if proposerIndex := fuluBlock.GetProposerIndex(); proposerIndex != nil {
			b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
		} else {
			b.ProposerIndex.Append(0)
		}

		b.ParentRoot.Append([]byte(fuluBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(fuluBlock.GetStateRoot()))

		body := fuluBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadElectra(body.GetExecutionPayload())

		return
	}

	// Unknown block version - append zeros.
	b.Slot.Append(0)
	b.ProposerIndex.Append(0)
	b.ParentRoot.Append(nil)
	b.StateRoot.Append(nil)
	b.Eth1DataBlockHash.Append(nil)
	b.Eth1DataDepositRoot.Append(nil)
	b.appendNullExecutionPayload()
}

func (b *canonicalBeaconBlockBatch) appendEth1Data(eth1Data *ethv1.Eth1Data) {
	if eth1Data == nil {
		b.Eth1DataBlockHash.Append(nil)
		b.Eth1DataDepositRoot.Append(nil)

		return
	}

	b.Eth1DataBlockHash.Append([]byte(eth1Data.GetBlockHash()))
	b.Eth1DataDepositRoot.Append([]byte(eth1Data.GetDepositRoot()))
}

func (b *canonicalBeaconBlockBatch) appendNullExecutionPayload() {
	b.ExecutionPayloadBlockHash.Append(proto.Nullable[[]byte]{})
	b.ExecutionPayloadBlockNumber.Append(proto.Nullable[uint32]{})
	b.ExecutionPayloadFeeRecipient.Append(proto.Nullable[string]{})
	b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadStateRoot.Append(proto.Nullable[[]byte]{})
	b.ExecutionPayloadParentHash.Append(proto.Nullable[[]byte]{})
}

//nolint:gosec // G115
func (b *canonicalBeaconBlockBatch) appendExecutionPayloadV2(payload *ethv1.ExecutionPayloadV2) {
	if payload == nil {
		b.appendNullExecutionPayload()

		return
	}

	b.ExecutionPayloadBlockHash.Append(proto.NewNullable[[]byte]([]byte(payload.GetBlockHash())))
	b.ExecutionPayloadFeeRecipient.Append(proto.NewNullable[string](payload.GetFeeRecipient()))
	b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBaseFeePerGas())))
	b.ExecutionPayloadStateRoot.Append(proto.NewNullable[[]byte]([]byte(payload.GetStateRoot())))
	b.ExecutionPayloadParentHash.Append(proto.NewNullable[[]byte]([]byte(payload.GetParentHash())))

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.ExecutionPayloadBlockNumber.Append(proto.NewNullable[uint32](uint32(blockNumber.GetValue())))
	} else {
		b.ExecutionPayloadBlockNumber.Append(proto.Nullable[uint32]{})
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

//nolint:gosec // G115
func (b *canonicalBeaconBlockBatch) appendExecutionPayloadCapellaV2(payload *ethv1.ExecutionPayloadCapellaV2) {
	if payload == nil {
		b.appendNullExecutionPayload()

		return
	}

	b.ExecutionPayloadBlockHash.Append(proto.NewNullable[[]byte]([]byte(payload.GetBlockHash())))
	b.ExecutionPayloadFeeRecipient.Append(proto.NewNullable[string](payload.GetFeeRecipient()))
	b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBaseFeePerGas())))
	b.ExecutionPayloadStateRoot.Append(proto.NewNullable[[]byte]([]byte(payload.GetStateRoot())))
	b.ExecutionPayloadParentHash.Append(proto.NewNullable[[]byte]([]byte(payload.GetParentHash())))

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.ExecutionPayloadBlockNumber.Append(proto.NewNullable[uint32](uint32(blockNumber.GetValue())))
	} else {
		b.ExecutionPayloadBlockNumber.Append(proto.Nullable[uint32]{})
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

//nolint:gosec // G115
func (b *canonicalBeaconBlockBatch) appendExecutionPayloadDeneb(payload *ethv1.ExecutionPayloadDeneb) {
	if payload == nil {
		b.appendNullExecutionPayload()

		return
	}

	b.ExecutionPayloadBlockHash.Append(proto.NewNullable[[]byte]([]byte(payload.GetBlockHash())))
	b.ExecutionPayloadFeeRecipient.Append(proto.NewNullable[string](payload.GetFeeRecipient()))
	b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBaseFeePerGas())))
	b.ExecutionPayloadStateRoot.Append(proto.NewNullable[[]byte]([]byte(payload.GetStateRoot())))
	b.ExecutionPayloadParentHash.Append(proto.NewNullable[[]byte]([]byte(payload.GetParentHash())))

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.ExecutionPayloadBlockNumber.Append(proto.NewNullable[uint32](uint32(blockNumber.GetValue())))
	} else {
		b.ExecutionPayloadBlockNumber.Append(proto.Nullable[uint32]{})
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

//nolint:gosec // G115
func (b *canonicalBeaconBlockBatch) appendExecutionPayloadElectra(payload *ethv1.ExecutionPayloadElectra) {
	if payload == nil {
		b.appendNullExecutionPayload()

		return
	}

	b.ExecutionPayloadBlockHash.Append(proto.NewNullable[[]byte]([]byte(payload.GetBlockHash())))
	b.ExecutionPayloadFeeRecipient.Append(proto.NewNullable[string](payload.GetFeeRecipient()))
	b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBaseFeePerGas())))
	b.ExecutionPayloadStateRoot.Append(proto.NewNullable[[]byte]([]byte(payload.GetStateRoot())))
	b.ExecutionPayloadParentHash.Append(proto.NewNullable[[]byte]([]byte(payload.GetParentHash())))

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.ExecutionPayloadBlockNumber.Append(proto.NewNullable[uint32](uint32(blockNumber.GetValue())))
	} else {
		b.ExecutionPayloadBlockNumber.Append(proto.Nullable[uint32]{})
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

//nolint:gosec // G115
func (b *canonicalBeaconBlockBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockV2()
	if additional == nil {
		b.BlockVersion.Append("")
		b.BlockRoot.Append(nil)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.SlotStartDateTime.Append(time.Time{})
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})

		return
	}

	b.BlockVersion.Append(additional.GetVersion())
	b.BlockRoot.Append([]byte(additional.GetBlockRoot()))

	if epoch := additional.GetEpoch(); epoch != nil {
		if epochNumber := epoch.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue()))
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

	if slot := additional.GetSlot(); slot != nil {
		// Note: Slot is already appended by appendPayload from the block's per-fork field.
		// Do NOT append b.Slot here to avoid column double-append.
		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.SlotStartDateTime.Append(time.Time{})
	}

	if totalBytes := additional.GetTotalBytes(); totalBytes != nil {
		b.BlockTotalBytes.Append(proto.NewNullable[uint32](uint32(totalBytes.GetValue())))
	} else {
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if totalBytesCompressed := additional.GetTotalBytesCompressed(); totalBytesCompressed != nil {
		b.BlockTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(totalBytesCompressed.GetValue())))
	} else {
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}

	if txCount := additional.GetTransactionsCount(); txCount != nil {
		b.ExecutionPayloadTransactionsCount.Append(proto.NewNullable[uint32](uint32(txCount.GetValue())))
	} else {
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
	}

	if txTotalBytes := additional.GetTransactionsTotalBytes(); txTotalBytes != nil {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.NewNullable[uint32](uint32(txTotalBytes.GetValue())))
	} else {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if txCompressed := additional.GetTransactionsTotalBytesCompressed(); txCompressed != nil {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(txCompressed.GetValue())))
	} else {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}
}
