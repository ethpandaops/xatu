package v1

import (
	"context"
	"fmt"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	BeaconValidatorsDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS
)

type BeaconValidatorsDeriverConfig struct {
	Enabled   bool                                 `yaml:"enabled" default:"true"`
	ChunkSize int                                  `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type BeaconValidatorsDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconValidatorsDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBeaconValidatorsDeriver(log logrus.FieldLogger, config *BeaconValidatorsDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconValidatorsDeriver {
	return &BeaconValidatorsDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/validators",
			"type":   BeaconValidatorsDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconValidatorsDeriver) CannonType() xatu.CannonType {
	return BeaconValidatorsDeriverName
}

func (b *BeaconValidatorsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (b *BeaconValidatorsDeriver) Name() string {
	return BeaconValidatorsDeriverName.String()
}

func (b *BeaconValidatorsDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconValidatorsDeriver) Start(ctx context.Context) error {
	b.log.WithFields(logrus.Fields{
		"chunk_size": b.cfg.ChunkSize,
		"enabled":    b.cfg.Enabled,
	}).Info("Starting BeaconValidatorsDeriver")

	if !b.cfg.Enabled {
		b.log.Info("Validator states deriver disabled")

		return nil
	}

	b.log.Info("Validator states deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *BeaconValidatorsDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconValidatorsDeriver) run(rctx context.Context) {
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

				events, slot, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).WithField("epoch", position.Next).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return err
				}

				b.lookAhead(ctx, position.LookAheads)

				// Be a good citizen and clean up the validator cache for the current epoch
				b.beacon.DeleteValidatorsFromCache(xatuethv1.SlotAsString(slot))

				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return errors.Wrap(err, "failed to send events")
					}
				}

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

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (b *BeaconValidatorsDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"BeaconValidatorsDeriver.lookAhead",
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		// Add the state to the preload queue so it's available when we need it
		b.beacon.LazyLoadValidators(xatuethv1.SlotAsString(phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))))
	}
}

func (b *BeaconValidatorsDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, phase0.Slot, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconValidatorsDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	spec, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(spec.SlotsPerEpoch))

	validatorsMap, err := b.beacon.GetValidators(ctx, xatuethv1.SlotAsString(boundarySlot))
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to fetch validator states")
	}

	// Chunk the validators per the configured chunk size
	chunkSize := b.cfg.ChunkSize

	var validatorChunks [][]*apiv1.Validator

	currentChunk := []*apiv1.Validator{}

	for _, validator := range validatorsMap {
		if len(currentChunk) == chunkSize {
			validatorChunks = append(validatorChunks, currentChunk)
			currentChunk = []*apiv1.Validator{}
		}

		currentChunk = append(currentChunk, validator)
	}

	if len(currentChunk) > 0 {
		validatorChunks = append(validatorChunks, currentChunk)
	}

	allEvents := []*xatu.DecoratedEvent{}

	for chunkNum, chunk := range validatorChunks {
		event, err := b.createEventFromValidators(ctx, chunk, epoch)
		if err != nil {
			b.log.
				WithError(err).
				WithField("chunk_size", len(chunk)).
				WithField("chunk_number", chunkNum).
				WithField("epoch", epoch).
				Error("Failed to create event from validator state")

			return nil, 0, err
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, boundarySlot, nil
}

func (b *BeaconValidatorsDeriver) createEventFromValidators(ctx context.Context, validators []*apiv1.Validator, epoch phase0.Epoch) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	data := xatu.Validators{}
	for _, validator := range validators {
		data.Validators = append(data.Validators, &xatuethv1.Validator{
			Index:   wrapperspb.UInt64(uint64(validator.Index)),
			Balance: wrapperspb.UInt64(uint64(validator.Balance)),
			Status:  wrapperspb.String(validator.Status.String()),
			Data: &xatuethv1.ValidatorData{
				Pubkey:                     wrapperspb.String(validator.Validator.PublicKey.String()),
				WithdrawalCredentials:      wrapperspb.String(fmt.Sprintf("%#x", validator.Validator.WithdrawalCredentials)),
				EffectiveBalance:           wrapperspb.UInt64(uint64(validator.Validator.EffectiveBalance)),
				Slashed:                    wrapperspb.Bool(validator.Validator.Slashed),
				ActivationEpoch:            wrapperspb.UInt64(uint64(validator.Validator.ActivationEpoch)),
				ActivationEligibilityEpoch: wrapperspb.UInt64(uint64(validator.Validator.ActivationEligibilityEpoch)),
				ExitEpoch:                  wrapperspb.UInt64(uint64(validator.Validator.ExitEpoch)),
				WithdrawableEpoch:          wrapperspb.UInt64(uint64(validator.Validator.WithdrawableEpoch)),
			},
		})
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1Validators{
			EthV1Validators: &data,
		},
	}

	additionalData, err := b.getAdditionalData(ctx, epoch)
	if err != nil {
		b.log.WithError(err).Error("Failed to get extra validator state data")

		return nil, err
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1Validators{
			EthV1Validators: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (b *BeaconValidatorsDeriver) getAdditionalData(_ context.Context, epoch phase0.Epoch) (*xatu.ClientMeta_AdditionalEthV1ValidatorsData, error) {
	epochInfo := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: uint64(epoch)},
			StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
		},
	}, nil
}
