package beacon

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV2BeaconBlockEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV2BeaconBlockTableName,
		beaconApiEthV2BeaconBlockEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV2BeaconBlockBatch() },
	))
}

func (b *beaconApiEthV2BeaconBlockBatch) FlattenTo(
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

func (b *beaconApiEthV2BeaconBlockBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV2BeaconBlockBatch) appendPayload(event *xatu.DecoratedEvent) {
	eventBlock := event.GetEthV2BeaconBlockV2()
	if eventBlock == nil {
		b.Slot.Append(0)
		b.ParentRoot.Append(nil)
		b.StateRoot.Append(nil)
		b.ProposerIndex.Append(0)
		b.Eth1DataBlockHash.Append(nil)
		b.Eth1DataDepositRoot.Append(nil)
		b.ExecutionPayloadBlockHash.Append(nil)
		b.ExecutionPayloadBlockNumber.Append(0)
		b.ExecutionPayloadFeeRecipient.Append("")
		b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
		b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})
		b.ExecutionPayloadStateRoot.Append(nil)
		b.ExecutionPayloadParentHash.Append(nil)

		return
	}

	b.appendPayloadFromEventBlockV2(eventBlock)
}

func (b *beaconApiEthV2BeaconBlockBatch) appendPayloadFromEventBlockV2(
	eventBlock *ethv2.EventBlockV2,
) {
	if phase0Block := eventBlock.GetPhase0Block(); phase0Block != nil {
		if slot := phase0Block.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.ParentRoot.Append([]byte(phase0Block.GetParentRoot()))
		b.StateRoot.Append([]byte(phase0Block.GetStateRoot()))

		if pi := phase0Block.GetProposerIndex(); pi != nil {
			b.ProposerIndex.Append(uint32(pi.GetValue())) //nolint:gosec // proposer index fits uint32
		} else {
			b.ProposerIndex.Append(0)
		}

		b.appendEth1Data(phase0Block.GetBody().GetEth1Data())
		b.appendNoExecutionPayload()

		return
	}

	if altairBlock := eventBlock.GetAltairBlock(); altairBlock != nil {
		if slot := altairBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.ParentRoot.Append([]byte(altairBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(altairBlock.GetStateRoot()))

		if pi := altairBlock.GetProposerIndex(); pi != nil {
			b.ProposerIndex.Append(uint32(pi.GetValue())) //nolint:gosec // proposer index fits uint32
		} else {
			b.ProposerIndex.Append(0)
		}

		b.appendEth1Data(altairBlock.GetBody().GetEth1Data())
		b.appendNoExecutionPayload()

		return
	}

	if bellatrixBlock := eventBlock.GetBellatrixBlock(); bellatrixBlock != nil {
		if slot := bellatrixBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.ParentRoot.Append([]byte(bellatrixBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(bellatrixBlock.GetStateRoot()))

		if pi := bellatrixBlock.GetProposerIndex(); pi != nil {
			b.ProposerIndex.Append(uint32(pi.GetValue())) //nolint:gosec // proposer index fits uint32
		} else {
			b.ProposerIndex.Append(0)
		}

		body := bellatrixBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadV2(body.GetExecutionPayload())

		return
	}

	if capellaBlock := eventBlock.GetCapellaBlock(); capellaBlock != nil {
		if slot := capellaBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.ParentRoot.Append([]byte(capellaBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(capellaBlock.GetStateRoot()))

		if pi := capellaBlock.GetProposerIndex(); pi != nil {
			b.ProposerIndex.Append(uint32(pi.GetValue())) //nolint:gosec // proposer index fits uint32
		} else {
			b.ProposerIndex.Append(0)
		}

		body := capellaBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadCapellaV2(body.GetExecutionPayload())

		return
	}

	if denebBlock := eventBlock.GetDenebBlock(); denebBlock != nil {
		if slot := denebBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.ParentRoot.Append([]byte(denebBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(denebBlock.GetStateRoot()))

		if pi := denebBlock.GetProposerIndex(); pi != nil {
			b.ProposerIndex.Append(uint32(pi.GetValue())) //nolint:gosec // proposer index fits uint32
		} else {
			b.ProposerIndex.Append(0)
		}

		body := denebBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadDeneb(body.GetExecutionPayload())

		return
	}

	if electraBlock := eventBlock.GetElectraBlock(); electraBlock != nil {
		if slot := electraBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.ParentRoot.Append([]byte(electraBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(electraBlock.GetStateRoot()))

		if pi := electraBlock.GetProposerIndex(); pi != nil {
			b.ProposerIndex.Append(uint32(pi.GetValue())) //nolint:gosec // proposer index fits uint32
		} else {
			b.ProposerIndex.Append(0)
		}

		body := electraBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadElectra(body.GetExecutionPayload())

		return
	}

	if fuluBlock := eventBlock.GetFuluBlock(); fuluBlock != nil {
		if slot := fuluBlock.GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
		} else {
			b.Slot.Append(0)
		}

		b.ParentRoot.Append([]byte(fuluBlock.GetParentRoot()))
		b.StateRoot.Append([]byte(fuluBlock.GetStateRoot()))

		if pi := fuluBlock.GetProposerIndex(); pi != nil {
			b.ProposerIndex.Append(uint32(pi.GetValue())) //nolint:gosec // proposer index fits uint32
		} else {
			b.ProposerIndex.Append(0)
		}

		body := fuluBlock.GetBody()
		b.appendEth1Data(body.GetEth1Data())
		b.appendExecutionPayloadElectra(body.GetExecutionPayload())

		return
	}

	// Unknown block type: append zero values.
	b.Slot.Append(0)
	b.ParentRoot.Append(nil)
	b.StateRoot.Append(nil)
	b.ProposerIndex.Append(0)
	b.appendEth1Data(nil)
	b.appendNoExecutionPayload()
}

func (b *beaconApiEthV2BeaconBlockBatch) appendEth1Data(eth1Data *ethv1.Eth1Data) {
	if eth1Data == nil {
		b.Eth1DataBlockHash.Append(nil)
		b.Eth1DataDepositRoot.Append(nil)

		return
	}

	b.Eth1DataBlockHash.Append([]byte(eth1Data.GetBlockHash()))
	b.Eth1DataDepositRoot.Append([]byte(eth1Data.GetDepositRoot()))
}

func (b *beaconApiEthV2BeaconBlockBatch) appendNoExecutionPayload() {
	b.ExecutionPayloadBlockHash.Append(nil)
	b.ExecutionPayloadBlockNumber.Append(0)
	b.ExecutionPayloadFeeRecipient.Append("")
	b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	b.ExecutionPayloadBlobGasUsed.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadExcessBlobGas.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadGasLimit.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadGasUsed.Append(proto.Nullable[uint64]{})
	b.ExecutionPayloadStateRoot.Append(nil)
	b.ExecutionPayloadParentHash.Append(nil)
}

func (b *beaconApiEthV2BeaconBlockBatch) appendExecutionPayloadV2(
	payload *ethv1.ExecutionPayloadV2,
) {
	if payload == nil {
		b.appendNoExecutionPayload()

		return
	}

	b.ExecutionPayloadBlockHash.Append([]byte(payload.GetBlockHash()))
	b.ExecutionPayloadFeeRecipient.Append(payload.GetFeeRecipient())
	b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBaseFeePerGas())))
	b.ExecutionPayloadStateRoot.Append([]byte(payload.GetStateRoot()))
	b.ExecutionPayloadParentHash.Append([]byte(payload.GetParentHash()))

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

func (b *beaconApiEthV2BeaconBlockBatch) appendExecutionPayloadCapellaV2(
	payload *ethv1.ExecutionPayloadCapellaV2,
) {
	if payload == nil {
		b.appendNoExecutionPayload()

		return
	}

	b.ExecutionPayloadBlockHash.Append([]byte(payload.GetBlockHash()))
	b.ExecutionPayloadFeeRecipient.Append(payload.GetFeeRecipient())
	b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBaseFeePerGas())))
	b.ExecutionPayloadStateRoot.Append([]byte(payload.GetStateRoot()))
	b.ExecutionPayloadParentHash.Append([]byte(payload.GetParentHash()))

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

func (b *beaconApiEthV2BeaconBlockBatch) appendExecutionPayloadDeneb(
	payload *ethv1.ExecutionPayloadDeneb,
) {
	if payload == nil {
		b.appendNoExecutionPayload()

		return
	}

	b.ExecutionPayloadBlockHash.Append([]byte(payload.GetBlockHash()))
	b.ExecutionPayloadFeeRecipient.Append(payload.GetFeeRecipient())
	b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBaseFeePerGas())))
	b.ExecutionPayloadStateRoot.Append([]byte(payload.GetStateRoot()))
	b.ExecutionPayloadParentHash.Append([]byte(payload.GetParentHash()))

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

func (b *beaconApiEthV2BeaconBlockBatch) appendExecutionPayloadElectra(
	payload *ethv1.ExecutionPayloadElectra,
) {
	if payload == nil {
		b.appendNoExecutionPayload()

		return
	}

	b.ExecutionPayloadBlockHash.Append([]byte(payload.GetBlockHash()))
	b.ExecutionPayloadFeeRecipient.Append(payload.GetFeeRecipient())
	b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(payload.GetBaseFeePerGas())))
	b.ExecutionPayloadStateRoot.Append([]byte(payload.GetStateRoot()))
	b.ExecutionPayloadParentHash.Append([]byte(payload.GetParentHash()))

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

func (b *beaconApiEthV2BeaconBlockBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Epoch.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockRoot.Append(nil)
		b.BlockVersion.Append("")
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})

		return
	}

	additionalV2 := event.GetMeta().GetClient().GetEthV2BeaconBlockV2()
	if additionalV2 == nil {
		b.Epoch.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockRoot.Append(nil)
		b.BlockVersion.Append("")
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})

		return
	}

	if epochNumber := additionalV2.GetEpoch().GetNumber(); epochNumber != nil {
		b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // epoch fits uint32
	} else {
		b.Epoch.Append(0)
	}

	if slot := additionalV2.GetSlot(); slot != nil {
		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.SlotStartDateTime.Append(time.Time{})
	}

	if epoch := additionalV2.GetEpoch(); epoch != nil {
		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.EpochStartDateTime.Append(time.Time{})
	}

	b.BlockRoot.Append([]byte(additionalV2.GetBlockRoot()))
	b.BlockVersion.Append(additionalV2.GetVersion())

	if totalBytes := additionalV2.GetTotalBytes(); totalBytes != nil {
		b.BlockTotalBytes.Append(proto.NewNullable[uint32](uint32(totalBytes.GetValue()))) //nolint:gosec // byte count fits uint32
	} else {
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if totalBytesCompressed := additionalV2.GetTotalBytesCompressed(); totalBytesCompressed != nil {
		b.BlockTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(totalBytesCompressed.GetValue()))) //nolint:gosec // byte count fits uint32
	} else {
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}

	if transactionsCount := additionalV2.GetTransactionsCount(); transactionsCount != nil {
		b.ExecutionPayloadTransactionsCount.Append(proto.NewNullable[uint32](uint32(transactionsCount.GetValue()))) //nolint:gosec // tx count fits uint32
	} else {
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
	}

	if transactionsTotalBytes := additionalV2.GetTransactionsTotalBytes(); transactionsTotalBytes != nil {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.NewNullable[uint32](uint32(transactionsTotalBytes.GetValue()))) //nolint:gosec // byte count fits uint32
	} else {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if v := additionalV2.GetTransactionsTotalBytesCompressed(); v != nil {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(v.GetValue()))) //nolint:gosec // byte count fits uint32
	} else {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}
}
