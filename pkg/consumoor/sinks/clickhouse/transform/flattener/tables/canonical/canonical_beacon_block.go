package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// canonicalBlockFields holds the extracted fields from an EventBlockV2 fork
// variant used by canonical block routes.
type canonicalBlockFields struct {
	BlockVersion    string
	Slot            uint64
	ProposerIndex   uint64
	ParentRoot      string
	StateRoot       string
	Eth1BlockHash   string
	Eth1DepositRoot string
	EPBlockHash     string
	EPBlockNumber   uint64
	EPFeeRecipient  string
	EPBaseFeePerGas string
	EPBlobGasUsed   *uint64
	EPExcessBlobGas *uint64
	EPGasLimit      *uint64
	EPGasUsed       *uint64
	EPStateRoot     string
	EPParentHash    string
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconBlockBatch()
		},
	))
}

func (b *canonicalBeaconBlockBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV2BeaconBlockV2()
	if payload == nil {
		return fmt.Errorf("nil EthV2BeaconBlockV2 payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockV2()

	f := extractCanonicalBlockFields(payload)

	b.UpdatedDateTime.Append(time.Now())
	b.Slot.Append(uint32(f.Slot)) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(addl.GetBlockRoot()))
	b.BlockVersion.Append(f.BlockVersion)

	if addl.GetTotalBytes() != nil {
		b.BlockTotalBytes.Append(proto.NewNullable[uint32](uint32(addl.GetTotalBytes().GetValue()))) //nolint:gosec // G115: total bytes fits uint32.
	} else {
		b.BlockTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if addl.GetTotalBytesCompressed() != nil {
		b.BlockTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(addl.GetTotalBytesCompressed().GetValue()))) //nolint:gosec // G115: total bytes compressed fits uint32.
	} else {
		b.BlockTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}

	b.ParentRoot.Append([]byte(f.ParentRoot))
	b.StateRoot.Append([]byte(f.StateRoot))
	b.ProposerIndex.Append(uint32(f.ProposerIndex)) //nolint:gosec // G115: proposer index fits uint32.
	b.Eth1DataBlockHash.Append([]byte(f.Eth1BlockHash))
	b.Eth1DataDepositRoot.Append([]byte(f.Eth1DepositRoot))

	// Execution payload fields are fully nullable in the canonical table.
	if f.EPBlockHash != "" {
		b.ExecutionPayloadBlockHash.Append(proto.NewNullable[[]byte]([]byte(f.EPBlockHash)))
	} else {
		b.ExecutionPayloadBlockHash.Append(proto.Nullable[[]byte]{})
	}

	if f.EPBlockHash != "" {
		b.ExecutionPayloadBlockNumber.Append(proto.NewNullable[uint32](uint32(f.EPBlockNumber))) //nolint:gosec // G115: block number fits uint32.
	} else {
		b.ExecutionPayloadBlockNumber.Append(proto.Nullable[uint32]{})
	}

	if f.EPFeeRecipient != "" {
		b.ExecutionPayloadFeeRecipient.Append(proto.NewNullable[string](f.EPFeeRecipient))
	} else {
		b.ExecutionPayloadFeeRecipient.Append(proto.Nullable[string]{})
	}

	if f.EPBaseFeePerGas != "" {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.NewNullable[proto.UInt128](flattener.ParseUInt128(f.EPBaseFeePerGas)))
	} else {
		b.ExecutionPayloadBaseFeePerGas.Append(proto.Nullable[proto.UInt128]{})
	}

	appendCanonicalNullableUInt64(b.ExecutionPayloadBlobGasUsed, f.EPBlobGasUsed)
	appendCanonicalNullableUInt64(b.ExecutionPayloadExcessBlobGas, f.EPExcessBlobGas)
	appendCanonicalNullableUInt64(b.ExecutionPayloadGasLimit, f.EPGasLimit)
	appendCanonicalNullableUInt64(b.ExecutionPayloadGasUsed, f.EPGasUsed)

	if f.EPStateRoot != "" {
		b.ExecutionPayloadStateRoot.Append(proto.NewNullable[[]byte]([]byte(f.EPStateRoot)))
	} else {
		b.ExecutionPayloadStateRoot.Append(proto.Nullable[[]byte]{})
	}

	if f.EPParentHash != "" {
		b.ExecutionPayloadParentHash.Append(proto.NewNullable[[]byte]([]byte(f.EPParentHash)))
	} else {
		b.ExecutionPayloadParentHash.Append(proto.Nullable[[]byte]{})
	}

	if addl.GetTransactionsCount() != nil {
		b.ExecutionPayloadTransactionsCount.Append(proto.NewNullable[uint32](uint32(addl.GetTransactionsCount().GetValue()))) //nolint:gosec // G115: tx count fits uint32.
	} else {
		b.ExecutionPayloadTransactionsCount.Append(proto.Nullable[uint32]{})
	}

	if addl.GetTransactionsTotalBytes() != nil {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.NewNullable[uint32](uint32(addl.GetTransactionsTotalBytes().GetValue()))) //nolint:gosec // G115: tx total bytes fits uint32.
	} else {
		b.ExecutionPayloadTransactionsTotalBytes.Append(proto.Nullable[uint32]{})
	}

	if addl.GetTransactionsTotalBytesCompressed() != nil {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.NewNullable[uint32](uint32(addl.GetTransactionsTotalBytesCompressed().GetValue()))) //nolint:gosec // G115: tx total bytes compressed fits uint32.
	} else {
		b.ExecutionPayloadTransactionsTotalBytesCompressed.Append(proto.Nullable[uint32]{})
	}

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV2BeaconBlockV2()

	if payload.GetMessage() == nil {
		return fmt.Errorf("nil block message (no fork variant): %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockV2()
	if addl == nil {
		return fmt.Errorf("nil EthV2BeaconBlockV2 additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}

// appendCanonicalNullableUInt64 appends a uint64 pointer as a nullable column
// entry. A nil pointer appends an unset nullable.
func appendCanonicalNullableUInt64(col *proto.ColNullable[uint64], v *uint64) {
	if v != nil {
		col.Append(proto.NewNullable[uint64](*v))
	} else {
		col.Append(proto.Nullable[uint64]{})
	}
}

// extractCanonicalBlockFields extracts common fields from any fork variant of
// EventBlockV2 for the canonical block table.
func extractCanonicalBlockFields(payload *v2.EventBlockV2) canonicalBlockFields {
	var f canonicalBlockFields

	switch msg := payload.GetMessage().(type) {
	case *v2.EventBlockV2_Phase0Block:
		blk := msg.Phase0Block
		f.BlockVersion = "phase0"
		f.Slot = blk.GetSlot().GetValue()
		f.ProposerIndex = blk.GetProposerIndex().GetValue()
		f.ParentRoot = blk.GetParentRoot()
		f.StateRoot = blk.GetStateRoot()
		f.Eth1BlockHash = blk.GetBody().GetEth1Data().GetBlockHash()
		f.Eth1DepositRoot = blk.GetBody().GetEth1Data().GetDepositRoot()

	case *v2.EventBlockV2_AltairBlock:
		blk := msg.AltairBlock
		f.BlockVersion = "altair"
		f.Slot = blk.GetSlot().GetValue()
		f.ProposerIndex = blk.GetProposerIndex().GetValue()
		f.ParentRoot = blk.GetParentRoot()
		f.StateRoot = blk.GetStateRoot()
		f.Eth1BlockHash = blk.GetBody().GetEth1Data().GetBlockHash()
		f.Eth1DepositRoot = blk.GetBody().GetEth1Data().GetDepositRoot()

	case *v2.EventBlockV2_BellatrixBlock:
		blk := msg.BellatrixBlock
		f.BlockVersion = "bellatrix"
		f.Slot = blk.GetSlot().GetValue()
		f.ProposerIndex = blk.GetProposerIndex().GetValue()
		f.ParentRoot = blk.GetParentRoot()
		f.StateRoot = blk.GetStateRoot()
		f.Eth1BlockHash = blk.GetBody().GetEth1Data().GetBlockHash()
		f.Eth1DepositRoot = blk.GetBody().GetEth1Data().GetDepositRoot()
		fillEPBasic(&f, blk.GetBody().GetExecutionPayload())

	case *v2.EventBlockV2_CapellaBlock:
		blk := msg.CapellaBlock
		f.BlockVersion = "capella"
		f.Slot = blk.GetSlot().GetValue()
		f.ProposerIndex = blk.GetProposerIndex().GetValue()
		f.ParentRoot = blk.GetParentRoot()
		f.StateRoot = blk.GetStateRoot()
		f.Eth1BlockHash = blk.GetBody().GetEth1Data().GetBlockHash()
		f.Eth1DepositRoot = blk.GetBody().GetEth1Data().GetDepositRoot()
		fillEPCapella(&f, blk.GetBody().GetExecutionPayload())

	case *v2.EventBlockV2_DenebBlock:
		blk := msg.DenebBlock
		f.BlockVersion = "deneb"
		f.Slot = blk.GetSlot().GetValue()
		f.ProposerIndex = blk.GetProposerIndex().GetValue()
		f.ParentRoot = blk.GetParentRoot()
		f.StateRoot = blk.GetStateRoot()
		f.Eth1BlockHash = blk.GetBody().GetEth1Data().GetBlockHash()
		f.Eth1DepositRoot = blk.GetBody().GetEth1Data().GetDepositRoot()
		fillEPDeneb(&f, blk.GetBody().GetExecutionPayload())

	case *v2.EventBlockV2_ElectraBlock:
		blk := msg.ElectraBlock
		f.BlockVersion = "electra"
		f.Slot = blk.GetSlot().GetValue()
		f.ProposerIndex = blk.GetProposerIndex().GetValue()
		f.ParentRoot = blk.GetParentRoot()
		f.StateRoot = blk.GetStateRoot()
		f.Eth1BlockHash = blk.GetBody().GetEth1Data().GetBlockHash()
		f.Eth1DepositRoot = blk.GetBody().GetEth1Data().GetDepositRoot()
		fillEPElectra(&f, blk.GetBody().GetExecutionPayload())

	case *v2.EventBlockV2_FuluBlock:
		blk := msg.FuluBlock
		f.BlockVersion = "fulu"
		f.Slot = blk.GetSlot().GetValue()
		f.ProposerIndex = blk.GetProposerIndex().GetValue()
		f.ParentRoot = blk.GetParentRoot()
		f.StateRoot = blk.GetStateRoot()
		f.Eth1BlockHash = blk.GetBody().GetEth1Data().GetBlockHash()
		f.Eth1DepositRoot = blk.GetBody().GetEth1Data().GetDepositRoot()
		fillEPElectra(&f, blk.GetBody().GetExecutionPayload())

	default:
		f.BlockVersion = "unknown"
	}

	return f
}

func fillEPBasic(f *canonicalBlockFields, ep *v1.ExecutionPayloadV2) {
	if ep == nil {
		return
	}

	f.EPBlockHash = ep.GetBlockHash()
	f.EPBlockNumber = ep.GetBlockNumber().GetValue()
	f.EPFeeRecipient = ep.GetFeeRecipient()
	f.EPBaseFeePerGas = ep.GetBaseFeePerGas()
	f.EPStateRoot = ep.GetStateRoot()
	f.EPParentHash = ep.GetParentHash()

	if ep.GetGasLimit() != nil {
		v := ep.GetGasLimit().GetValue()
		f.EPGasLimit = &v
	}

	if ep.GetGasUsed() != nil {
		v := ep.GetGasUsed().GetValue()
		f.EPGasUsed = &v
	}
}

func fillEPCapella(f *canonicalBlockFields, ep *v1.ExecutionPayloadCapellaV2) {
	if ep == nil {
		return
	}

	f.EPBlockHash = ep.GetBlockHash()
	f.EPBlockNumber = ep.GetBlockNumber().GetValue()
	f.EPFeeRecipient = ep.GetFeeRecipient()
	f.EPBaseFeePerGas = ep.GetBaseFeePerGas()
	f.EPStateRoot = ep.GetStateRoot()
	f.EPParentHash = ep.GetParentHash()

	if ep.GetGasLimit() != nil {
		v := ep.GetGasLimit().GetValue()
		f.EPGasLimit = &v
	}

	if ep.GetGasUsed() != nil {
		v := ep.GetGasUsed().GetValue()
		f.EPGasUsed = &v
	}
}

func fillEPDeneb(f *canonicalBlockFields, ep *v1.ExecutionPayloadDeneb) {
	if ep == nil {
		return
	}

	f.EPBlockHash = ep.GetBlockHash()
	f.EPBlockNumber = ep.GetBlockNumber().GetValue()
	f.EPFeeRecipient = ep.GetFeeRecipient()
	f.EPBaseFeePerGas = ep.GetBaseFeePerGas()
	f.EPStateRoot = ep.GetStateRoot()
	f.EPParentHash = ep.GetParentHash()

	if ep.GetGasLimit() != nil {
		v := ep.GetGasLimit().GetValue()
		f.EPGasLimit = &v
	}

	if ep.GetGasUsed() != nil {
		v := ep.GetGasUsed().GetValue()
		f.EPGasUsed = &v
	}

	if ep.GetBlobGasUsed() != nil {
		v := ep.GetBlobGasUsed().GetValue()
		f.EPBlobGasUsed = &v
	}

	if ep.GetExcessBlobGas() != nil {
		v := ep.GetExcessBlobGas().GetValue()
		f.EPExcessBlobGas = &v
	}
}

func fillEPElectra(f *canonicalBlockFields, ep *v1.ExecutionPayloadElectra) {
	if ep == nil {
		return
	}

	f.EPBlockHash = ep.GetBlockHash()
	f.EPBlockNumber = ep.GetBlockNumber().GetValue()
	f.EPFeeRecipient = ep.GetFeeRecipient()
	f.EPBaseFeePerGas = ep.GetBaseFeePerGas()
	f.EPStateRoot = ep.GetStateRoot()
	f.EPParentHash = ep.GetParentHash()

	if ep.GetGasLimit() != nil {
		v := ep.GetGasLimit().GetValue()
		f.EPGasLimit = &v
	}

	if ep.GetGasUsed() != nil {
		v := ep.GetGasUsed().GetValue()
		f.EPGasUsed = &v
	}

	if ep.GetBlobGasUsed() != nil {
		v := ep.GetBlobGasUsed().GetValue()
		f.EPBlobGasUsed = &v
	}

	if ep.GetExcessBlobGas() != nil {
		v := ep.GetExcessBlobGas().GetValue()
		f.EPExcessBlobGas = &v
	}
}
