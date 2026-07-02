package v1

import (
	"context"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	eth2client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
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
	RandaoDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_RANDAO
)

type RandaoDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type RandaoDeriver struct {
	log               observability.ContextualLogger
	cfg               *RandaoDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewRandaoDeriver(log observability.ContextualLogger, config *RandaoDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *RandaoDeriver {
	return &RandaoDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/randao",
			"type":   RandaoDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *RandaoDeriver) CannonType() xatu.CannonType {
	return RandaoDeriverName
}

func (b *RandaoDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (b *RandaoDeriver) Name() string {
	return RandaoDeriverName.String()
}

func (b *RandaoDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *RandaoDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Randao deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Randao deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *RandaoDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *RandaoDeriver) run(rctx context.Context) {
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

func (b *RandaoDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"RandaoDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))), //nolint:gosec // epoch fits int64
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))
	stateID := xatuethv1.SlotAsString(boundarySlot)

	randao, err := b.fetchRandao(ctx, stateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch randao")
	}

	event, err := b.createEventFromRandao(ctx, epoch, stateID, randao)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create event from randao")
	}

	return []*xatu.DecoratedEvent{event}, nil
}

func (b *RandaoDeriver) fetchRandao(ctx context.Context, stateID string) (phase0.Root, error) {
	provider, isProvider := b.beacon.Node().Service().(eth2client.BeaconStateRandaoProvider)
	if !isProvider {
		return phase0.Root{}, errors.New("beacon state randao provider not available")
	}

	rsp, err := provider.BeaconStateRandao(ctx, &api.BeaconStateRandaoOpts{
		State: stateID,
	})
	if err != nil {
		return phase0.Root{}, err
	}

	if rsp == nil || rsp.Data == nil {
		return phase0.Root{}, errors.New("no randao returned")
	}

	return *rsp.Data, nil
}

func (b *RandaoDeriver) createEventFromRandao(ctx context.Context, epoch phase0.Epoch, stateID string, randao phase0.Root) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_RANDAO,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconStateRandao{
			EthV1BeaconStateRandao: &xatu.RandaoData{
				Randao: randao.String(),
			},
		},
	}

	additionalData := b.getAdditionalData(ctx, epoch, stateID)

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStateRandao{
		EthV1BeaconStateRandao: additionalData,
	}

	return decoratedEvent, nil
}

func (b *RandaoDeriver) getAdditionalData(_ context.Context, epoch phase0.Epoch, stateID string) *xatu.ClientMeta_AdditionalEthV1BeaconStateRandaoData {
	epochInfo := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1BeaconStateRandaoData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: uint64(epoch)},
			StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
		},
		StateId: stateID,
	}
}
