package v1

import (
	"context"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	FinalityCheckpointDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT
)

type FinalityCheckpointDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type FinalityCheckpointDeriver struct {
	log               observability.ContextualLogger
	cfg               *FinalityCheckpointDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewFinalityCheckpointDeriver(log observability.ContextualLogger, config *FinalityCheckpointDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *FinalityCheckpointDeriver {
	return &FinalityCheckpointDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/finality_checkpoint",
			"type":   FinalityCheckpointDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *FinalityCheckpointDeriver) CannonType() xatu.CannonType {
	return FinalityCheckpointDeriverName
}

func (b *FinalityCheckpointDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (b *FinalityCheckpointDeriver) Name() string {
	return FinalityCheckpointDeriverName.String()
}

func (b *FinalityCheckpointDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *FinalityCheckpointDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Finality checkpoint deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Finality checkpoint deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *FinalityCheckpointDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *FinalityCheckpointDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position.
				position, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).WithField("epoch", position.Next).WithContext(ctx).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					b.log.WithError(err).WithField("next_attempt", timer).WithContext(rctx).Warn("Failed to process")
				}),
			}

			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				b.log.WithError(err).WithContext(rctx).Warn("Failed to process")
			}
		}
	}
}

func (b *FinalityCheckpointDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"FinalityCheckpointDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))), //nolint:gosec // epoch fits int64
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))
	stateID := xatuethv1.SlotAsString(boundarySlot)

	finality, err := b.fetchFinality(ctx, stateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch finality checkpoints")
	}

	event, err := b.createEventFromFinality(ctx, finality, epoch, stateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create event from finality checkpoints")
	}

	return []*xatu.DecoratedEvent{event}, nil
}

func (b *FinalityCheckpointDeriver) fetchFinality(ctx context.Context, stateID string) (*apiv1.Finality, error) {
	provider, isProvider := b.beacon.Node().Service().(client.FinalityProvider)
	if !isProvider {
		return nil, errors.New("finality provider not found")
	}

	rsp, err := provider.Finality(ctx, &api.FinalityOpts{
		State: stateID,
	})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (b *FinalityCheckpointDeriver) createEventFromFinality(ctx context.Context, finality *apiv1.Finality, epoch phase0.Epoch, stateID string) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	data := &xatu.FinalityCheckpointData{
		PreviousJustified: checkpointToProto(finality.PreviousJustified),
		CurrentJustified:  checkpointToProto(finality.Justified),
		Finalized:         checkpointToProto(finality.Finalized),
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconStateFinalityCheckpoint{
			EthV1BeaconStateFinalityCheckpoint: data,
		},
	}

	additionalData, err := b.getAdditionalData(ctx, epoch, stateID)
	if err != nil {
		b.log.WithError(err).WithContext(ctx).Error("Failed to get extra finality checkpoint data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStateFinalityCheckpoint{
		EthV1BeaconStateFinalityCheckpoint: additionalData,
	}

	return decoratedEvent, nil
}

func (b *FinalityCheckpointDeriver) getAdditionalData(_ context.Context, epoch phase0.Epoch, stateID string) (*xatu.ClientMeta_AdditionalEthV1BeaconStateFinalityCheckpointData, error) {
	epochInfo := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1BeaconStateFinalityCheckpointData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: uint64(epoch)},
			StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
		},
		StateId: stateID,
	}, nil
}

func checkpointToProto(checkpoint *phase0.Checkpoint) *xatuethv1.Checkpoint {
	if checkpoint == nil {
		return nil
	}

	return &xatuethv1.Checkpoint{
		Epoch: uint64(checkpoint.Epoch),
		Root:  fmt.Sprintf("%#x", checkpoint.Root),
	}
}
