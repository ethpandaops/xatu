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
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	xatuethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	BLSToExecutionChangeDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE
)

type BLSToExecutionChangeDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

type BLSToExecutionChangeDeriver struct {
	log               logrus.FieldLogger
	cfg               *BLSToExecutionChangeDeriverConfig
	iterator          *iterator.CheckpointIterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBLSToExecutionChangeDeriver(log logrus.FieldLogger, config *BLSToExecutionChangeDeriverConfig, iter *iterator.CheckpointIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BLSToExecutionChangeDeriver {
	return &BLSToExecutionChangeDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/bls_to_execution_change"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BLSToExecutionChangeDeriver) CannonType() xatu.CannonType {
	return BLSToExecutionChangeDeriverName
}

func (b *BLSToExecutionChangeDeriver) Name() string {
	return BLSToExecutionChangeDeriverName.String()
}

func (b *BLSToExecutionChangeDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BLSToExecutionChangeDeriver) ActivationFork() string {
	return ethereum.ForkNameCapella
}

func (b *BLSToExecutionChangeDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("BLS to execution change deriver disabled")

		return nil
	}

	b.log.Info("BLS to execution change deriver enabled")

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *BLSToExecutionChangeDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BLSToExecutionChangeDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() error {
				ctx, span := observability.Tracer().Start(rctx, fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					return err
				}

				// Get the next slot
				location, lookAheads, err := b.iterator.Next(ctx)
				if err != nil {
					return err
				}

				// Look ahead
				b.lookAheadAtLocation(ctx, lookAheads)

				// Process the epoch
				events, err := b.processEpoch(ctx, phase0.Epoch(location.GetEthV2BeaconBlockBlsToExecutionChange().GetEpoch()))
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

			if err := backoff.RetryNotify(operation, bo, func(err error, timer time.Duration) {
				b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
			}); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAheadAtLocation takes the upcoming locations and looks ahead to do any pre-processing that might be required.
func (b *BLSToExecutionChangeDeriver) lookAheadAtLocation(ctx context.Context, locations []*xatu.CannonLocation) {
	_, span := observability.Tracer().Start(ctx,
		"BLSToExecutionChangeDeriver.lookAheadAtLocations",
	)
	defer span.End()

	if locations == nil {
		return
	}

	for _, location := range locations {
		// Get the next look ahead epoch
		epoch := phase0.Epoch(location.GetEthV2BeaconBlockBlsToExecutionChange().GetEpoch())

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

func (b *BLSToExecutionChangeDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BLSToExecutionChangeDeriver.processEpoch",
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

func (b *BLSToExecutionChangeDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BLSToExecutionChangeDeriver.processSlot",
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

	blockIdentifier, err := GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	events := []*xatu.DecoratedEvent{}

	changes, err := b.getBLSToExecutionChanges(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		event, err := b.createEvent(ctx, change, blockIdentifier)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for BLS to execution change %s", change.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *BLSToExecutionChangeDeriver) getBLSToExecutionChanges(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv2.SignedBLSToExecutionChangeV2, error) {
	changes := []*xatuethv2.SignedBLSToExecutionChangeV2{}

	chs, err := block.BLSToExecutionChanges()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain BLS to execution changes")
	}

	for _, change := range chs {
		changes = append(changes, &xatuethv2.SignedBLSToExecutionChangeV2{
			Message: &xatuethv2.BLSToExecutionChangeV2{
				ValidatorIndex:     wrapperspb.UInt64(uint64(change.Message.ValidatorIndex)),
				FromBlsPubkey:      change.Message.FromBLSPubkey.String(),
				ToExecutionAddress: change.Message.ToExecutionAddress.String(),
			},
			Signature: change.Signature.String(),
		})
	}

	return changes, nil
}

func (b *BLSToExecutionChangeDeriver) createEvent(ctx context.Context, change *xatuethv2.SignedBLSToExecutionChangeV2, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: change,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockBlsToExecutionChange{
		EthV2BeaconBlockBlsToExecutionChange: &xatu.ClientMeta_AdditionalEthV2BeaconBlockBLSToExecutionChangeData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
