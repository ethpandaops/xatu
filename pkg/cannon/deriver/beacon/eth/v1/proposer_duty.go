package v1

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ProposerDutyDeriverName = xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY
)

type ProposerDutyDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type ProposerDutyDeriver struct {
	log               logrus.FieldLogger
	cfg               *ProposerDutyDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewProposerDutyDeriver(log logrus.FieldLogger, config *ProposerDutyDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ProposerDutyDeriver {
	return &ProposerDutyDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/proposer_duty",
			"type":   ProposerDutyDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *ProposerDutyDeriver) CannonType() xatu.CannonType {
	return ProposerDutyDeriverName
}

func (b *ProposerDutyDeriver) ActivationFork() string {
	return ethereum.ForkNamePhase0
}

func (b *ProposerDutyDeriver) Name() string {
	return ProposerDutyDeriverName.String()
}

func (b *ProposerDutyDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *ProposerDutyDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Proposer duty deriver disabled")

		return nil
	}

	b.log.Info("Proposer duty deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *ProposerDutyDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *ProposerDutyDeriver) run(rctx context.Context) {
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

				// Get the next position.
				position, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					span.SetStatus(codes.Error, err.Error())

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

func (b *ProposerDutyDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ProposerDutyDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	// Get the proposer duties for this epoch
	proposerDuties, err := b.beacon.Node().FetchProposerDuties(ctx, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch proposer duties")
	}

	allEvents := []*xatu.DecoratedEvent{}

	for _, duty := range proposerDuties {
		event, err := b.createEventFromProposerDuty(ctx, duty)
		if err != nil {
			b.log.
				WithError(err).
				WithField("slot", duty.Slot).
				WithField("epoch", epoch).
				Error("Failed to create event from proposer duty")

			return nil, err
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (b *ProposerDutyDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ProposerDutyDeriver.lookAhead",
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

func (b *ProposerDutyDeriver) createEventFromProposerDuty(ctx context.Context, duty *apiv1.ProposerDuty) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1ProposerDuty{
			EthV1ProposerDuty: &xatuethv1.ProposerDuty{
				Slot:           wrapperspb.UInt64(uint64(duty.Slot)),
				Pubkey:         fmt.Sprintf("0x%s", hex.EncodeToString(duty.PubKey[:])),
				ValidatorIndex: wrapperspb.UInt64(uint64(duty.ValidatorIndex)),
			},
		},
	}

	additionalData, err := b.getAdditionalData(ctx, duty)
	if err != nil {
		b.log.WithError(err).Error("Failed to get extra proposer duty data")

		return nil, err
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1ProposerDuty{
			EthV1ProposerDuty: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (b *ProposerDutyDeriver) getAdditionalData(_ context.Context, duty *apiv1.ProposerDuty) (*xatu.ClientMeta_AdditionalEthV1ProposerDutyData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1ProposerDutyData{
		StateId: xatuethv1.StateIDFinalized,
	}

	slot := b.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(duty.Slot))
	epoch := b.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(duty.Slot))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(duty.Slot)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
