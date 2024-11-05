package event

import (
	"context"
	"math/big"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/eth"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	ssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ValidatorBeaconBlock struct {
	log        logrus.FieldLogger
	snapshot   *ValidatorBeaconBlockDataSnapshot
	event      *api.VersionedProposal
	beacon     *ethereum.BeaconNode
	clientMeta *xatu.ClientMeta
	id         uuid.UUID
}

type ValidatorBeaconBlockDataSnapshot struct {
	RequestAt       time.Time
	RequestDuration time.Duration
}

func NewValidatorBeaconBlock(log logrus.FieldLogger, event *api.VersionedProposal, snapshot *ValidatorBeaconBlockDataSnapshot, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ValidatorBeaconBlock {
	return &ValidatorBeaconBlock{
		log:        log.WithField("event", "BEACON_API_ETH_V3_VALIDATOR_BEACON_BLOCK"),
		event:      event,
		snapshot:   snapshot,
		beacon:     beacon,
		clientMeta: clientMeta,
		id:         uuid.New(),
	}
}

func (e *ValidatorBeaconBlock) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	data, err := eth.NewEventBlockV2FromVersionSignedBeaconBlock(e.event)
	if err != nil {
		return nil, err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BEACON_BLOCK,
			DateTime: timestamppb.New(e.snapshot.RequestAt),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV3ValidatorBeaconBlock{
			EthV3ValidatorBeaconBlock: data,
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra beacon block data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV3ValidatorBeaconBlock{
			EthV3ValidatorBeaconBlock: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *ValidatorBeaconBlock) ShouldIgnore(_ context.Context) (bool, error) {
	if e.event == nil {
		return true, nil
	}

	// @TODO(matty): Many other ingesters will ignore blocks >16 slots old. Needed for validator beacon block event?

	return false, nil
}

func (e *ValidatorBeaconBlock) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV3ValidatorBeaconBlockData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV3ValidatorBeaconBlockData{}

	slotI, err := e.event.Slot()
	if err != nil {
		return nil, err
	}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(slotI))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(slotI))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(slotI)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Version = e.event.Version.String()
	extra.RequestedAt = timestamppb.New(e.snapshot.RequestAt)
	extra.RequestDurationMs = &wrapperspb.UInt64Value{Value: uint64(e.snapshot.RequestDuration.Milliseconds())}

	// extra.BlockRoot = e.blockRoot @TODO(matty): Ask Sammo about this, as its proposed, not sure.

	var (
		txCount, txSize                  int
		transactionsBytes                []byte
		totalBytes, totalBytesCompressed uint64
	)

	addTxData := func(txs [][]byte) {
		txCount = len(txs)

		for _, tx := range txs {
			txSize += len(tx)
			transactionsBytes = append(transactionsBytes, tx...)
		}
	}

	switch e.event.Version {
	case spec.DataVersionPhase0:
		totalBytes, totalBytesCompressed, err = computeBlockSize(e.event.Phase0.Body)
		if err != nil {
			e.log.WithError(err).Warn("Failed to compute phase0 block size")
		}
	case spec.DataVersionAltair:
		totalBytes, totalBytesCompressed, err = computeBlockSize(e.event.Altair.Body)
		if err != nil {
			e.log.WithError(err).Warn("Failed to compute altair block size")
		}
	case spec.DataVersionBellatrix:
		totalBytes, totalBytesCompressed, err = computeBlockSize(e.event.Bellatrix.Body)
		if err != nil {
			e.log.WithError(err).Warn("Failed to compute bellatrix block size")
		}

		bellatrixTxs := make([][]byte, len(e.event.Bellatrix.Body.ExecutionPayload.Transactions))
		for i, tx := range e.event.Bellatrix.Body.ExecutionPayload.Transactions {
			bellatrixTxs[i] = tx
		}

		addTxData(bellatrixTxs)
	case spec.DataVersionCapella:
		totalBytes, totalBytesCompressed, err = computeBlockSize(e.event.Capella.Body)
		if err != nil {
			e.log.WithError(err).Warn("Failed to compute capella block size")
		}

		capellaTxs := make([][]byte, len(e.event.Capella.Body.ExecutionPayload.Transactions))
		for i, tx := range e.event.Capella.Body.ExecutionPayload.Transactions {
			capellaTxs[i] = tx
		}

		addTxData(capellaTxs)
	case spec.DataVersionDeneb:
		totalBytes, totalBytesCompressed, err = computeBlockSize(e.event.Deneb.Block.Body)
		if err != nil {
			e.log.WithError(err).Warn("Failed to compute deneb block size")
		}

		denebTxs := make([][]byte, len(e.event.Deneb.Block.Body.ExecutionPayload.Transactions))
		for i, tx := range e.event.Deneb.Block.Body.ExecutionPayload.Transactions {
			denebTxs[i] = tx
		}

		addTxData(denebTxs)
	default:
		e.log.WithError(err).Warn("Failed to get block message to compute block size. Missing fork version?")
	}

	extra.TotalBytes = wrapperspb.UInt64(totalBytes)
	extra.TotalBytesCompressed = wrapperspb.UInt64(totalBytesCompressed)
	extra.TransactionsCount = wrapperspb.UInt64(uint64(txCount))
	extra.TransactionsTotalBytes = wrapperspb.UInt64(uint64(txSize))
	extra.TransactionsTotalBytesCompressed = wrapperspb.UInt64(uint64(len(snappy.Encode(nil, transactionsBytes))))
	extra.ExecutionValue = new(big.Int).SetUint64(e.event.ExecutionValue.Uint64()).String()
	extra.ConsensusValue = new(big.Int).SetUint64(e.event.ConsensusValue.Uint64()).String()

	return extra, nil
}

func computeBlockSize(body ssz.Marshaler) (uint64, uint64, error) {
	sszData, err := ssz.MarshalSSZ(body)
	if err != nil {
		return 0, 0, errors.New("Failed to marshal (SSZ) block message")
	}

	dataSize := len(sszData)
	compressedDataSize := len(snappy.Encode(nil, sszData))

	return uint64(dataSize), uint64(compressedDataSize), nil
}
