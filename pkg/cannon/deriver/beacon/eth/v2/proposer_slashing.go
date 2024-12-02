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
	ProposerSlashingDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING
)

type ProposerSlashingDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type ProposerSlashingDeriver struct {
	log               logrus.FieldLogger
	cfg               *ProposerSlashingDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewProposerSlashingDeriver(log logrus.FieldLogger, config *ProposerSlashingDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ProposerSlashingDeriver {
	return &ProposerSlashingDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v2/proposer_slashing",
			"type":   ProposerSlashingDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *ProposerSlashingDeriver) CannonType() xatu.CannonType {
	return ProposerSlashingDeriverName
}

func (b *ProposerSlashingDeriver) Name() string {
	return ProposerSlashingDeriverName.String()
}

func (b *ProposerSlashingDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (b *ProposerSlashingDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *ProposerSlashingDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Proposer slashing deriver disabled")

		return nil
	}

	b.log.Info("Proposer slashing deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *ProposerSlashingDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *ProposerSlashingDeriver) run(rctx context.Context) {
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

				// Get the next position
				position, err := b.iterator.Next(ctx)
				if err != nil {
					return err
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					return err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						return errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
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

func (b *ProposerSlashingDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ProposerSlashingDeriver.processEpoch",
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

func (b *ProposerSlashingDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ProposerSlashingDeriver.processSlot",
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

	slashings, err := b.getProposerSlashings(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, slashing := range slashings {
		event, err := b.createEvent(ctx, slashing, blockIdentifier)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for proposer slashing %s", slashing.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *ProposerSlashingDeriver) getProposerSlashings(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.ProposerSlashingV2, error) {
	slashings := []*xatuethv1.ProposerSlashingV2{}

	blockSlashings, err := block.ProposerSlashings()
	if err != nil {
		return nil, err
	}

	for _, slashing := range blockSlashings {
		slashings = append(slashings, &xatuethv1.ProposerSlashingV2{
			SignedHeader_1: &xatuethv1.SignedBeaconBlockHeaderV2{
				Message: &xatuethv1.BeaconBlockHeaderV2{
					Slot:          wrapperspb.UInt64(uint64(slashing.SignedHeader1.Message.Slot)),
					ProposerIndex: wrapperspb.UInt64(uint64(slashing.SignedHeader1.Message.ProposerIndex)),
					ParentRoot:    slashing.SignedHeader1.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader1.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader1.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader1.Signature.String(),
			},
			SignedHeader_2: &xatuethv1.SignedBeaconBlockHeaderV2{
				Message: &xatuethv1.BeaconBlockHeaderV2{
					Slot:          wrapperspb.UInt64(uint64(slashing.SignedHeader2.Message.Slot)),
					ProposerIndex: wrapperspb.UInt64(uint64(slashing.SignedHeader2.Message.ProposerIndex)),
					ParentRoot:    slashing.SignedHeader2.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader2.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader2.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader2.Signature.String(),
			},
		})
	}

	return slashings, nil
}

// lookAhead attempts to pre-load any blocks that might be required for the epochs that are coming up.
func (b *ProposerSlashingDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ProposerSlashingDeriver.lookAhead",
	)
	defer span.End()

	if epochs == nil {
		return
	}

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

func (b *ProposerSlashingDeriver) createEvent(ctx context.Context, slashing *xatuethv1.ProposerSlashingV2, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: slashing,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockProposerSlashing{
		EthV2BeaconBlockProposerSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockProposerSlashingData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
