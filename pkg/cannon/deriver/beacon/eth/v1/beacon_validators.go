package v1

import (
	"context"
	"fmt"
	"time"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
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
	BeaconValidatorsDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS
)

type BeaconValidatorsDeriverConfig struct {
	Enabled  bool `yaml:"enabled" default:"true"`
	ChunkSize int  `yaml:"chunkSize" default:"100"`
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
		log:        log.WithField("module", "cannon/event/beacon/eth/v1/validators"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconValidatorsDeriver) CannonType() xatu.CannonType {
	return BeaconValidatorsDeriverName
}

func (b *BeaconValidatorsDeriver) ActivationFork() string {
	return ethereum.ForkNamePhase0
}

func (b *BeaconValidatorsDeriver) Name() string {
	return BeaconValidatorsDeriverName.String()
}

func (b *BeaconValidatorsDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconValidatorsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Validator states deriver disabled")

		return nil
	}

	b.log.Info("Validator states deriver enabled")

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

				// Get the next location.
				location, _, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				finality, err := b.beacon.Node().Finality()
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return errors.Wrap(err, "failed to get finality")
				}

				if finality == nil {
					span.SetStatus(codes.Error, "finality is nil")

					return errors.New("finality is nil")
				}

				checkpoint := finality.Finalized

				processingFinalizedEpoch := false
				epochToProcess := location.GetEthV1BeaconValidators().GetBackfillingCheckpointMarker().GetBackfillEpoch() - 1

				if checkpoint.Epoch > phase0.Epoch(location.GetEthV1BeaconValidators().BackfillingCheckpointMarker.FinalizedEpoch) {
					processingFinalizedEpoch = true
					epochToProcess = int64(location.GetEthV1BeaconValidators().BackfillingCheckpointMarker.FinalizedEpoch) + 1
				}

				if !processingFinalizedEpoch && epochToProcess == -1 {
					return nil
				}

				events, err := b.processEpoch(ctx, phase0.Epoch(epochToProcess))
				if err != nil {
					b.log.WithError(err).WithField("epoch", epochToProcess).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return err
				}

				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return errors.Wrap(err, "failed to send events")
					}
				}

				if processingFinalizedEpoch {
					location.Data = &xatu.CannonLocation_EthV1BeaconValidators{
						EthV1BeaconValidators: &xatu.CannonLocationEthV1BeaconValidators{
							BackfillingCheckpointMarker: &xatu.BackfillingCheckpointMarker{
								BackfillEpoch:  location.GetEthV1BeaconValidators().GetBackfillingCheckpointMarker().GetBackfillEpoch(),
								FinalizedEpoch: uint64(epochToProcess),
							},
						},
					}
				} else {
					location.Data = &xatu.CannonLocation_EthV1BeaconValidators{
						EthV1BeaconValidators: &xatu.CannonLocationEthV1BeaconValidators{
							BackfillingCheckpointMarker: &xatu.BackfillingCheckpointMarker{
								BackfillEpoch:  epochToProcess,
								FinalizedEpoch: location.GetEthV1BeaconValidators().GetBackfillingCheckpointMarker().GetFinalizedEpoch(),
							},
						},
					}
				}

				span.SetAttributes(attribute.Int64("epoch", epochToProcess))
				span.SetAttributes(attribute.Bool("is_backfill", !processingFinalizedEpoch))

				if err := b.iterator.UpdateLocation(ctx, location); err != nil {
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

func (b *BeaconValidatorsDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconValidatorsDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	validatorStates, err := b.getValidatorsClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch validator states")
	}

	spec, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(spec.SlotsPerEpoch))

	validatorsResponse, err := validatorStates.Validators(ctx, &api.ValidatorsOpts{
		State: xatuethv1.SlotAsString(boundarySlot),
		Common: api.CommonOpts{
			Timeout: 6 * time.Minute, // If we can't fetch 1 epoch of validators within 1 epoch we will never catch up.
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch validator states")
	}

	validatorsMap := validatorsResponse.Data

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

			return nil, err
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}

func (b *BeaconValidatorsDeriver) getValidatorsClient(ctx context.Context) (client.ValidatorsProvider, error) {
	if provider, isProvider := b.beacon.Node().Service().(client.ValidatorsProvider); isProvider {
		return provider, nil
	}

	return nil, errors.New("validator states client not found")
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
