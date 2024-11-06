package v3

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

type ValidatorBlock struct {
	log        logrus.FieldLogger
	snapshot   *ValidatorBlockDataSnapshot
	event      *api.VersionedProposal
	beacon     *ethereum.BeaconNode
	clientMeta *xatu.ClientMeta
	id         uuid.UUID
}

type ValidatorBlockDataSnapshot struct {
	RequestAt       time.Time
	RequestDuration time.Duration
}

func NewValidatorBlock(
	log logrus.FieldLogger,
	event *api.VersionedProposal,
	snapshot *ValidatorBlockDataSnapshot,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *ValidatorBlock {
	return &ValidatorBlock{
		log:        log.WithField("event", "BEACON_API_ETH_V3_VALIDATOR_BLOCK"),
		event:      event,
		snapshot:   snapshot,
		beacon:     beacon,
		clientMeta: clientMeta,
		id:         uuid.New(),
	}
}

func (e *ValidatorBlock) Decorate(_ context.Context) (*xatu.DecoratedEvent, error) {
	data, err := eth.NewEventBlockV2FromVersionedProposal(e.event)
	if err != nil {
		return nil, err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK,
			DateTime: timestamppb.New(e.snapshot.RequestAt),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV3ValidatorBlock{
			EthV3ValidatorBlock: data,
		},
	}

	if err := e.addAdditionalData(decoratedEvent); err != nil {
		e.log.WithError(err).Error("Failed to get extra beacon block data")
	}

	return decoratedEvent, nil
}

func (e *ValidatorBlock) ShouldIgnore(_ context.Context) (bool, error) {
	return e.event == nil, nil
}

func (e *ValidatorBlock) addAdditionalData(decoratedEvent *xatu.DecoratedEvent) error {
	additionalData, err := e.getAdditionalData()
	if err != nil {
		return err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV3ValidatorBlock{
		EthV3ValidatorBlock: additionalData,
	}

	return nil
}

func (e *ValidatorBlock) getAdditionalData() (*xatu.ClientMeta_AdditionalEthV3ValidatorBlockData, error) {
	proposalSlot, err := e.event.Slot()
	if err != nil {
		return nil, err
	}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(proposalSlot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(proposalSlot))

	extra := &xatu.ClientMeta_AdditionalEthV3ValidatorBlockData{
		Version:     e.event.Version.String(),
		RequestedAt: timestamppb.New(e.snapshot.RequestAt),
		RequestDurationMs: &wrapperspb.UInt64Value{
			Value: safeUint64FromInt64(e.snapshot.RequestDuration.Milliseconds()),
		},
		Slot: &xatu.SlotV2{
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
			Number:        &wrapperspb.UInt64Value{Value: uint64(proposalSlot)},
		},
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
	}

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
	extra.TransactionsCount = wrapperspb.UInt64(safeUint64FromInt64(int64(txCount)))
	extra.TransactionsTotalBytes = wrapperspb.UInt64(safeUint64FromInt64(int64(txSize)))
	extra.TransactionsTotalBytesCompressed = wrapperspb.UInt64(uint64(len(snappy.Encode(nil, transactionsBytes))))
	extra.ExecutionValue = new(big.Int).SetUint64(e.event.ExecutionValue.Uint64()).String()
	extra.ConsensusValue = new(big.Int).SetUint64(e.event.ConsensusValue.Uint64()).String()

	return extra, nil
}

func computeBlockSize(body ssz.Marshaler) (size, compressed uint64, err error) {
	sszData, err := ssz.MarshalSSZ(body)
	if err != nil {
		return size, compressed, errors.New("Failed to marshal (SSZ) block message")
	}

	size = uint64(len(sszData))
	compressed = uint64(len(snappy.Encode(nil, sszData)))

	return size, compressed, nil
}

func safeUint64FromInt64(value int64) uint64 {
	if value < 0 {
		return 0
	}

	return uint64(value)
}
