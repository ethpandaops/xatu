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
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	VoluntaryExitDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT
)

type VoluntaryExitDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

type VoluntaryExitDeriver struct {
	log               logrus.FieldLogger
	cfg               *VoluntaryExitDeriverConfig
	iterator          *iterator.CheckpointIterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewVoluntaryExitDeriver(log logrus.FieldLogger, config *VoluntaryExitDeriverConfig, iter *iterator.CheckpointIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *VoluntaryExitDeriver {
	return &VoluntaryExitDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/voluntary_exit"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *VoluntaryExitDeriver) CannonType() xatu.CannonType {
	return VoluntaryExitDeriverName
}

func (b *VoluntaryExitDeriver) Name() string {
	return VoluntaryExitDeriverName.String()
}

func (b *VoluntaryExitDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *VoluntaryExitDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Voluntary exit deriver disabled")

		return nil
	}

	b.log.Info("Voluntary exit deriver enabled")

	// Start our main loop
	go b.run(ctx)

	return nil
}

func (b *VoluntaryExitDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *VoluntaryExitDeriver) run(rctx context.Context) {
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
				location, lookAhead, err := b.iterator.Next(ctx)
				if err != nil {
					return err
				}

				// Look ahead
				b.lookAheadAtLocation(ctx, lookAhead)

				// Process the epoch
				events, err := b.processEpoch(ctx, phase0.Epoch(location.GetEthV2BeaconBlockVoluntaryExit().GetEpoch()))
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
func (b *VoluntaryExitDeriver) lookAheadAtLocation(ctx context.Context, locations []*xatu.CannonLocation) {
	_, span := observability.Tracer().Start(ctx,
		"VoluntaryExitDeriver.lookAheadAtLocations",
	)
	defer span.End()

	if locations == nil {
		return
	}

	for _, location := range locations {
		// Get the next look ahead epoch
		epoch := phase0.Epoch(location.GetEthV2BeaconBlockVoluntaryExit().GetEpoch())

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

func (b *VoluntaryExitDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"VoluntaryExitDeriver.processEpoch",
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

func (b *VoluntaryExitDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"VoluntaryExitDeriver.processSlot",
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

	exits, err := b.getVoluntaryExits(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, exit := range exits {
		event, err := b.createEvent(ctx, exit, blockIdentifier)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for voluntary exit %s", exit.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *VoluntaryExitDeriver) getVoluntaryExits(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.SignedVoluntaryExitV2, error) {
	exits := []*xatuethv1.SignedVoluntaryExitV2{}

	voluntaryExits, err := block.VoluntaryExits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain voluntary exits")
	}

	for _, exit := range voluntaryExits {
		exits = append(exits, &xatuethv1.SignedVoluntaryExitV2{
			Message: &xatuethv1.VoluntaryExitV2{
				Epoch:          wrapperspb.UInt64(uint64(exit.Message.Epoch)),
				ValidatorIndex: wrapperspb.UInt64(uint64(exit.Message.ValidatorIndex)),
			},
			Signature: exit.Signature.String(),
		})
	}

	return exits, nil
}

func (b *VoluntaryExitDeriver) createEvent(ctx context.Context, exit *xatuethv1.SignedVoluntaryExitV2, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: exit,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit{
		EthV2BeaconBlockVoluntaryExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockVoluntaryExitData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
