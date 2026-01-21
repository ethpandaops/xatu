package deriver

import (
	"context"
	"fmt"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
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
	BeaconValidatorsDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS
)

// BeaconValidatorsDeriverConfig holds the configuration for the BeaconValidatorsDeriver.
type BeaconValidatorsDeriverConfig struct {
	Enabled   bool `yaml:"enabled" default:"true"`
	ChunkSize int  `yaml:"chunkSize" default:"100"`
}

// BeaconValidatorsDeriver derives beacon validator state events from the consensus layer.
// It processes epochs and emits decorated events for validator states, chunked for efficiency.
type BeaconValidatorsDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconValidatorsDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewBeaconValidatorsDeriver creates a new BeaconValidatorsDeriver instance.
func NewBeaconValidatorsDeriver(
	log logrus.FieldLogger,
	config *BeaconValidatorsDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *BeaconValidatorsDeriver {
	return &BeaconValidatorsDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/beacon_validators",
			"type":   BeaconValidatorsDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (d *BeaconValidatorsDeriver) CannonType() xatu.CannonType {
	return BeaconValidatorsDeriverName
}

func (d *BeaconValidatorsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (d *BeaconValidatorsDeriver) Name() string {
	return BeaconValidatorsDeriverName.String()
}

func (d *BeaconValidatorsDeriver) OnEventsDerived(
	_ context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	d.onEventsCallbacks = append(d.onEventsCallbacks, fn)
}

func (d *BeaconValidatorsDeriver) Start(ctx context.Context) error {
	d.log.WithFields(logrus.Fields{
		"chunk_size": d.cfg.ChunkSize,
		"enabled":    d.cfg.Enabled,
	}).Info("Starting BeaconValidatorsDeriver")

	if !d.cfg.Enabled {
		d.log.Info("Validator states deriver disabled")

		return nil
	}

	d.log.Info("Validator states deriver enabled")

	if err := d.iterator.Start(ctx, d.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	d.run(ctx)

	return nil
}

func (d *BeaconValidatorsDeriver) Stop(_ context.Context) error {
	return nil
}

func (d *BeaconValidatorsDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", d.Name()),
					trace.WithAttributes(
						attribute.String("network", d.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := d.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position
				position, err := d.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Process the epoch
				events, slot, err := d.processEpoch(ctx, position.Epoch)
				if err != nil {
					d.log.WithError(err).WithField("epoch", position.Epoch).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				d.lookAhead(ctx, position.LookAheadEpochs)

				// Be a good citizen and clean up the validator cache for the current epoch
				d.beacon.DeleteValidatorsFromCache(xatuethv1.SlotAsString(slot))

				// Send the events
				for _, fn := range d.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := d.iterator.UpdateLocation(ctx, position); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					d.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				d.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (d *BeaconValidatorsDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"BeaconValidatorsDeriver.lookAhead",
	)
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		d.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		// Add the state to the preload queue so it's available when we need it
		d.beacon.LazyLoadValidators(xatuethv1.SlotAsString(phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))))
	}
}

func (d *BeaconValidatorsDeriver) processEpoch(
	ctx context.Context,
	epoch phase0.Epoch,
) ([]*xatu.DecoratedEvent, phase0.Slot, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconValidatorsDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))

	validatorsMap, err := d.beacon.GetValidators(ctx, xatuethv1.SlotAsString(boundarySlot))
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to fetch validator states")
	}

	// Chunk the validators per the configured chunk size
	chunkSize := d.cfg.ChunkSize

	var validatorChunks [][]*apiv1.Validator

	currentChunk := make([]*apiv1.Validator, 0, chunkSize)

	for _, validator := range validatorsMap {
		if len(currentChunk) == chunkSize {
			validatorChunks = append(validatorChunks, currentChunk)
			currentChunk = make([]*apiv1.Validator, 0, chunkSize)
		}

		currentChunk = append(currentChunk, validator)
	}

	if len(currentChunk) > 0 {
		validatorChunks = append(validatorChunks, currentChunk)
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, len(validatorChunks))

	for chunkNum, chunk := range validatorChunks {
		event, err := d.createEventFromValidators(ctx, chunk, epoch)
		if err != nil {
			d.log.
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

func (d *BeaconValidatorsDeriver) createEventFromValidators(
	ctx context.Context,
	validators []*apiv1.Validator,
	epoch phase0.Epoch,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := d.ctx.CreateClientMeta(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client metadata")
	}

	// Make a clone of the metadata
	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
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

	additionalData, err := d.getAdditionalData(epoch)
	if err != nil {
		d.log.WithError(err).Error("Failed to get extra validator state data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1Validators{
		EthV1Validators: additionalData,
	}

	return decoratedEvent, nil
}

func (d *BeaconValidatorsDeriver) getAdditionalData(
	epoch phase0.Epoch,
) (*xatu.ClientMeta_AdditionalEthV1ValidatorsData, error) {
	epochInfo := d.ctx.Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: uint64(epoch)},
			StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
		},
	}, nil
}
