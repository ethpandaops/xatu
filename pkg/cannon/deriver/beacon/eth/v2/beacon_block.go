package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/eth"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	ssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	BeaconBlockDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK
)

type BeaconBlockDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type BeaconBlockDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconBlockDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBeaconBlockDeriver(log logrus.FieldLogger, config *BeaconBlockDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconBlockDeriver {
	return &BeaconBlockDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v2/beacon_block",
			"type":   BeaconBlockDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconBlockDeriver) CannonType() xatu.CannonType {
	return BeaconBlockDeriverName
}

func (b *BeaconBlockDeriver) ActivationFork() string {
	return ethereum.ForkNamePhase0
}

func (b *BeaconBlockDeriver) Name() string {
	return BeaconBlockDeriverName.String()
}

func (b *BeaconBlockDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconBlockDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Beacon block deriver disabled")

		return nil
	}

	b.log.Info("Beacon block deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *BeaconBlockDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconBlockDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() error {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Get the next slot
				position, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				span.AddEvent("Location updated. Done.")

				bo.Reset()

				return nil
			}

			if err := backoff.RetryNotify(operation, bo, func(err error, timer time.Duration) {
				b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
			}); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (b *BeaconBlockDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"BeaconBlockDeriver.lookAhead",
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			b.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (b *BeaconBlockDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconBlockDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

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
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconBlockDeriver.processSlot",
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

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

	additionalData, err := b.getAdditionalData(ctx, block)
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

func (b *BeaconBlockDeriver) getAdditionalData(_ context.Context, block *spec.VersionedSignedBeaconBlock) (*xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data, error) {
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
	case spec.DataVersionDeneb:
		denebTxs := make([][]byte, len(block.Deneb.Message.Body.ExecutionPayload.Transactions))
		for i, tx := range block.Deneb.Message.Body.ExecutionPayload.Transactions {
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

	// Always set to true when derived from the cannon.
	extra.FinalizedWhenRequested = true

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
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version)
	}
}
