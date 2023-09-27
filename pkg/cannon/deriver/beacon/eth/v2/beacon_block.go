package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/proto/eth"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	xatuethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/golang/snappy"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	BeaconBlockDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK
)

type BeaconBlockDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

type BeaconBlockDeriver struct {
	log                 logrus.FieldLogger
	cfg                 *BeaconBlockDeriverConfig
	iterator            *iterator.CheckpointIterator
	onEventsCallbacks   []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	onLocationCallbacks []func(ctx context.Context, location uint64) error
	beacon              *ethereum.BeaconNode
	clientMeta          *xatu.ClientMeta
}

func NewBeaconBlockDeriver(log logrus.FieldLogger, config *BeaconBlockDeriverConfig, iter *iterator.CheckpointIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconBlockDeriver {
	return &BeaconBlockDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/beacon_block"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconBlockDeriver) CannonType() xatu.CannonType {
	return BeaconBlockDeriverName
}

func (b *BeaconBlockDeriver) Name() string {
	return BeaconBlockDeriverName.String()
}

func (b *BeaconBlockDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconBlockDeriver) OnLocationUpdated(ctx context.Context, fn func(ctx context.Context, location uint64) error) {
	b.onLocationCallbacks = append(b.onLocationCallbacks, fn)
}

func (b *BeaconBlockDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Beacon block deriver disabled")

		return nil
	}

	b.log.Info("Beacon block deriver enabled")

	// Start our main loop
	go b.run(ctx)

	return nil
}

func (b *BeaconBlockDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconBlockDeriver) run(ctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return
		default:
			operation := func() error {
				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					return err
				}

				// Get the next slot
				location, lookAhead, err := b.iterator.Next(ctx)
				if err != nil {
					return err
				}

				// Look ahead
				b.lookAheadAtLocation(ctx, lookAhead)

				for _, fn := range b.onLocationCallbacks {
					if errr := fn(ctx, location.GetEthV2BeaconBlock().GetEpoch()); errr != nil {
						b.log.WithError(errr).Error("Failed to send location")
					}
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, phase0.Epoch(location.GetEthV2BeaconBlock().GetEpoch()))
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					return err
				}

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						return errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, location); err != nil {
					return err
				}

				bo.Reset()

				return nil
			}

			if err := backoff.Retry(operation, bo); err != nil {
				b.log.WithError(err).Error("Failed to process location")
			}
		}
	}
}

// lookAheadAtLocation takes the upcoming locations and looks ahead to do any pre-processing that might be required.
func (b *BeaconBlockDeriver) lookAheadAtLocation(ctx context.Context, locations []*xatu.CannonLocation) {
	if locations == nil {
		return
	}

	for _, location := range locations {
		// Get the next look ahead epoch
		epoch := phase0.Epoch(location.GetEthV2BeaconBlock().GetEpoch())

		sp, err := b.beacon.Node().Spec()
		if err != nil {
			b.log.WithError(err).WithField("epoch", epoch).Warn("Failed to look ahead at epoch")

			return
		}

		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			b.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (b *BeaconBlockDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := []*xatu.DecoratedEvent{}

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := b.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (b *BeaconBlockDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	// Get the block
	block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	event, err := b.createEventFromBlock(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create event from block for slot %d", slot)
	}

	return []*xatu.DecoratedEvent{event}, nil
}

func (b *BeaconBlockDeriver) createEventFromBlock(ctx context.Context, block *spec.VersionedSignedBeaconBlock) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	data, err := eth.NewEventBlockV2FromVersionSignedBeaconBlock(block)
	if err != nil {
		return nil, err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: data,
		},
	}

	additionalData, err := b.getAdditionalData(ctx, block, data)
	if err != nil {
		b.log.WithError(err).Error("Failed to get extra beacon block data")

		return nil, err
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (b *BeaconBlockDeriver) getAdditionalData(_ context.Context, block *spec.VersionedSignedBeaconBlock, data *xatuethv2.EventBlockV2) (*xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data, error) {
	extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{}

	slotI, err := block.Slot()
	if err != nil {
		return nil, err
	}

	slot := b.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(slotI))
	epoch := b.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(slotI))

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

	addTxData := func(txs [][]byte) {
		txCount = len(txs)

		for _, tx := range txs {
			txSize += len(tx)
			transactionsBytes = append(transactionsBytes, tx...)
		}
	}

	dataAsJSON, err := json.Marshal(block)
	if err != nil {
		return nil, err
	}

	dataSize := len(dataAsJSON)
	compressedData := snappy.Encode(nil, dataAsJSON)
	compressedDataSize := len(compressedData)

	blockRoot, err := block.Root()
	if err != nil {
		return nil, err
	}

	extra.BlockRoot = fmt.Sprintf("%#x", blockRoot)

	switch block.Version {
	case spec.DataVersionPhase0:
	case spec.DataVersionBellatrix:
		bellatrixTxs := make([][]byte, len(block.Bellatrix.Message.Body.ExecutionPayload.Transactions))
		for i, tx := range block.Bellatrix.Message.Body.ExecutionPayload.Transactions {
			bellatrixTxs[i] = tx
		}

		addTxData(bellatrixTxs)
	case spec.DataVersionCapella:
		capellaTxs := make([][]byte, len(block.Capella.Message.Body.ExecutionPayload.Transactions))
		for i, tx := range block.Capella.Message.Body.ExecutionPayload.Transactions {
			capellaTxs[i] = tx
		}

		addTxData(capellaTxs)
	}

	compressedTransactions := snappy.Encode(nil, transactionsBytes)
	compressedTxSize := len(compressedTransactions)

	extra.TotalBytes = wrapperspb.UInt64(uint64(dataSize))
	extra.TotalBytesCompressed = wrapperspb.UInt64(uint64(compressedDataSize))
	extra.TransactionsCount = wrapperspb.UInt64(uint64(txCount))
	extra.TransactionsTotalBytes = wrapperspb.UInt64(uint64(txSize))
	extra.TransactionsTotalBytesCompressed = wrapperspb.UInt64(uint64(compressedTxSize))

	// Always set to true when derived from the cannon.
	extra.FinalizedWhenRequested = true

	return extra, nil
}
