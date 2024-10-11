package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/eth"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	ssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type BeaconBlock struct {
	log logrus.FieldLogger

	now time.Time

	blockRoot      string
	event          *spec.VersionedSignedBeaconBlock
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewBeaconBlock(log logrus.FieldLogger, blockRoot string, event *spec.VersionedSignedBeaconBlock, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *BeaconBlock {
	return &BeaconBlock{
		log:            log.WithField("event", "BEACON_API_ETH_V2_BEACON_BLOCK_V2"),
		now:            now,
		blockRoot:      blockRoot,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *BeaconBlock) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	data, err := eth.NewEventBlockV2FromVersionSignedBeaconBlock(e.event)
	if err != nil {
		return nil, err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: data,
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra beacon block data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *BeaconBlock) ShouldIgnore(ctx context.Context) (bool, error) {
	if e.event == nil {
		return true, nil
	}

	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"hash":                  hash,
			"time_since_first_item": time.Since(item.Value()),
			"slot":                  e.event.Slot,
		}).Debug("Duplicate beacon block event received")

		return true, nil
	}

	currentSlot, _, err := e.beacon.Node().Wallclock().Now()
	if err != nil {
		return true, err
	}

	// ignore blocks that are more than 16 slots old
	slotLimit := currentSlot.Number() - 16

	slot, err := e.event.Slot()
	if err != nil {
		return true, err
	}

	if uint64(slot) < slotLimit {
		return true, nil
	}

	return false, nil
}

func (e *BeaconBlock) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data, error) {
	extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{}

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
	extra.BlockRoot = e.blockRoot
	// Always set to false for sentry events
	extra.FinalizedWhenRequested = false

	var txCount int

	var txSize int

	var transactionsBytes []byte

	addTxData := func(txs [][]byte) {
		txCount = len(txs)

		for _, tx := range txs {
			txSize += len(tx)
			transactionsBytes = append(transactionsBytes, tx...)
		}
	}

	blockMessage, err := getBlockMessage(e.event)
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

	switch e.event.Version {
	case spec.DataVersionBellatrix:
		bellatrixTxs := make([][]byte, len(e.event.Bellatrix.Message.Body.ExecutionPayload.Transactions))
		for i, tx := range e.event.Bellatrix.Message.Body.ExecutionPayload.Transactions {
			bellatrixTxs[i] = tx
		}

		addTxData(bellatrixTxs)
	case spec.DataVersionCapella:
		capellaTxs := make([][]byte, len(e.event.Capella.Message.Body.ExecutionPayload.Transactions))
		for i, tx := range e.event.Capella.Message.Body.ExecutionPayload.Transactions {
			capellaTxs[i] = tx
		}

		addTxData(capellaTxs)
	case spec.DataVersionDeneb:
		denebTxs := make([][]byte, len(e.event.Deneb.Message.Body.ExecutionPayload.Transactions))
		for i, tx := range e.event.Deneb.Message.Body.ExecutionPayload.Transactions {
			denebTxs[i] = tx
		}

		addTxData(denebTxs)
	}

	compressedTransactions := snappy.Encode(nil, transactionsBytes)
	compressedTxSize := len(compressedTransactions)

	extra.TotalBytes = wrapperspb.UInt64(uint64(dataSize))
	extra.TotalBytesCompressed = wrapperspb.UInt64(uint64(compressedDataSize))
	extra.TransactionsCount = wrapperspb.UInt64(uint64(txCount))
	extra.TransactionsTotalBytes = wrapperspb.UInt64(uint64(txSize))
	extra.TransactionsTotalBytesCompressed = wrapperspb.UInt64(uint64(compressedTxSize))

	return extra, nil
}

func getBlockMessage(block *spec.VersionedSignedBeaconBlock) (ssz.Marshaler, error) {
	switch block.Version {
	case spec.DataVersionAltair:
		return block.Altair.Message, nil
	case spec.DataVersionBellatrix:
		return block.Bellatrix.Message, nil
	case spec.DataVersionCapella:
		return block.Capella.Message, nil
	case spec.DataVersionDeneb:
		return block.Deneb.Message, nil
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version)
	}
}
