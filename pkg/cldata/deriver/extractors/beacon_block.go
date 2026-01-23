package extractors

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	"github.com/ethpandaops/xatu/pkg/proto/eth"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	ssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "beacon_block",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractBeaconBlock,
	})
}

// ExtractBeaconBlock extracts a beacon block event from a block.
func ExtractBeaconBlock(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	_ cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	builder := deriver.NewEventBuilder(ctxProvider)

	event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2)
	if err != nil {
		return nil, err
	}

	data, err := eth.NewEventBlockV2FromVersionSignedBeaconBlock(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create event block")
	}

	event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockV2{
		EthV2BeaconBlockV2: data,
	}

	additionalData, err := getBeaconBlockAdditionalData(block, blockID, ctxProvider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get additional data")
	}

	event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockV2{
		EthV2BeaconBlockV2: additionalData,
	}

	return []*xatu.DecoratedEvent{event}, nil
}

func getBeaconBlockAdditionalData(
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	ctxProvider cldata.ContextProvider,
) (*xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data, error) {
	extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{}

	slotI, err := block.Slot()
	if err != nil {
		return nil, err
	}

	wallclock := ctxProvider.Wallclock()
	slot := wallclock.Slots().FromNumber(uint64(slotI))
	epoch := wallclock.Epochs().FromSlot(uint64(slotI))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(slotI)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Version = block.Version.String()

	var txCount int

	var txSize int

	var transactionsBytes []byte

	transactions, err := block.ExecutionTransactions()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get execution transactions")
	}

	txs := make([][]byte, len(transactions))
	for i, tx := range transactions {
		txs[i] = tx
	}

	txCount = len(txs)

	for _, tx := range txs {
		txSize += len(tx)
		transactionsBytes = append(transactionsBytes, tx...)
	}

	blockMessage, err := getBlockMessage(block)
	if err != nil {
		return nil, err
	}

	sszData, err := ssz.MarshalSSZ(blockMessage)
	if err != nil {
		return nil, err
	}

	dataSize := len(sszData)
	compressedData := snappy.Encode(nil, sszData)
	compressedDataSize := len(compressedData)

	blockRoot, err := block.Root()
	if err != nil {
		return nil, err
	}

	extra.BlockRoot = fmt.Sprintf("%#x", blockRoot)

	compressedTransactions := snappy.Encode(nil, transactionsBytes)
	compressedTxSize := len(compressedTransactions)

	extra.TotalBytes = wrapperspb.UInt64(uint64(dataSize))
	extra.TotalBytesCompressed = wrapperspb.UInt64(uint64(compressedDataSize))
	extra.TransactionsCount = wrapperspb.UInt64(uint64(txCount))
	//nolint:gosec // txSize is always non-negative
	extra.TransactionsTotalBytes = wrapperspb.UInt64(uint64(txSize))
	extra.TransactionsTotalBytesCompressed = wrapperspb.UInt64(uint64(compressedTxSize))

	// Always set to true when derived from the cannon.
	extra.FinalizedWhenRequested = true

	// Copy block identifier fields
	if blockID != nil {
		extra.Slot = blockID.Slot
		extra.Epoch = blockID.Epoch
	}

	return extra, nil
}

func getBlockMessage(block *spec.VersionedSignedBeaconBlock) (ssz.Marshaler, error) {
	switch block.Version {
	case spec.DataVersionPhase0:
		return block.Phase0.Message, nil
	case spec.DataVersionAltair:
		return block.Altair.Message, nil
	case spec.DataVersionBellatrix:
		return block.Bellatrix.Message, nil
	case spec.DataVersionCapella:
		return block.Capella.Message, nil
	case spec.DataVersionDeneb:
		return block.Deneb.Message, nil
	case spec.DataVersionElectra:
		return block.Electra.Message, nil
	case spec.DataVersionFulu:
		return block.Fulu.Message, nil
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version)
	}
}
