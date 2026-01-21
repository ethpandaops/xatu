package deriver

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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
	ProposerSlashingDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING
)

// ProposerSlashingDeriverConfig holds the configuration for the ProposerSlashingDeriver.
type ProposerSlashingDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// ProposerSlashingDeriver derives proposer slashing events from the consensus layer.
// It processes epochs of blocks and emits decorated events for each proposer slashing.
type ProposerSlashingDeriver struct {
	log               logrus.FieldLogger
	cfg               *ProposerSlashingDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewProposerSlashingDeriver creates a new ProposerSlashingDeriver instance.
func NewProposerSlashingDeriver(
	log logrus.FieldLogger,
	config *ProposerSlashingDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *ProposerSlashingDeriver {
	return &ProposerSlashingDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/proposer_slashing",
			"type":   ProposerSlashingDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (p *ProposerSlashingDeriver) CannonType() xatu.CannonType {
	return ProposerSlashingDeriverName
}

func (p *ProposerSlashingDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (p *ProposerSlashingDeriver) Name() string {
	return ProposerSlashingDeriverName.String()
}

func (p *ProposerSlashingDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	p.onEventsCallbacks = append(p.onEventsCallbacks, fn)
}

func (p *ProposerSlashingDeriver) Start(ctx context.Context) error {
	if !p.cfg.Enabled {
		p.log.Info("Proposer slashing deriver disabled")

		return nil
	}

	p.log.Info("Proposer slashing deriver enabled")

	if err := p.iterator.Start(ctx, p.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	p.run(ctx)

	return nil
}

func (p *ProposerSlashingDeriver) Stop(ctx context.Context) error {
	return nil
}

func (p *ProposerSlashingDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", p.Name()),
					trace.WithAttributes(
						attribute.String("network", p.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := p.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position
				position, err := p.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				p.lookAhead(ctx, position.LookAheadEpochs)

				// Process the epoch
				events, err := p.processEpoch(ctx, position.Epoch)
				if err != nil {
					p.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range p.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := p.iterator.UpdateLocation(ctx, position); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Location updated. Done.")

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					p.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				p.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (p *ProposerSlashingDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ProposerSlashingDeriver.lookAhead",
	)
	defer span.End()

	sp, err := p.beacon.Node().Spec()
	if err != nil {
		p.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			p.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (p *ProposerSlashingDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ProposerSlashingDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := p.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := p.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (p *ProposerSlashingDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ProposerSlashingDeriver.processSlot",
		//nolint:gosec // slot numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := p.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, p.ctx.Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	events := make([]*xatu.DecoratedEvent, 0)

	slashings, err := p.getProposerSlashings(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get proposer slashings for slot %d", slot)
	}

	for _, slashing := range slashings {
		event, err := p.createEvent(ctx, slashing, blockIdentifier)
		if err != nil {
			p.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for proposer slashing %s", slashing.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (p *ProposerSlashingDeriver) getProposerSlashings(
	_ context.Context,
	block *spec.VersionedSignedBeaconBlock,
) ([]*xatuethv1.ProposerSlashingV2, error) {
	slashings := make([]*xatuethv1.ProposerSlashingV2, 0)

	blockSlashings, err := block.ProposerSlashings()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain proposer slashings")
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

func (p *ProposerSlashingDeriver) createEvent(
	ctx context.Context,
	slashing *xatuethv1.ProposerSlashingV2,
	identifier *xatu.BlockIdentifier,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := p.ctx.CreateClientMeta(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client metadata")
	}

	// Make a clone of the metadata
	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
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
