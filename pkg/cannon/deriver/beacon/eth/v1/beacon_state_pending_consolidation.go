package v1

import (
	"context"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
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
	BeaconStatePendingConsolidationDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION
)

type BeaconStatePendingConsolidationDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type BeaconStatePendingConsolidationDeriver struct {
	log               observability.ContextualLogger
	cfg               *BeaconStatePendingConsolidationDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBeaconStatePendingConsolidationDeriver(log observability.ContextualLogger, config *BeaconStatePendingConsolidationDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconStatePendingConsolidationDeriver {
	return &BeaconStatePendingConsolidationDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/beacon_state_pending_consolidation",
			"type":   BeaconStatePendingConsolidationDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconStatePendingConsolidationDeriver) CannonType() xatu.CannonType {
	return BeaconStatePendingConsolidationDeriverName
}

func (b *BeaconStatePendingConsolidationDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionElectra
}

func (b *BeaconStatePendingConsolidationDeriver) Name() string {
	return BeaconStatePendingConsolidationDeriverName.String()
}

func (b *BeaconStatePendingConsolidationDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconStatePendingConsolidationDeriver) Start(ctx context.Context) error {
	b.log.WithField("enabled", b.cfg.Enabled).WithContext(ctx).Info("Starting BeaconStatePendingConsolidationDeriver")

	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Pending consolidation deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Pending consolidation deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx)

	return nil
}

func (b *BeaconStatePendingConsolidationDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconStatePendingConsolidationDeriver) run(rctx context.Context) {
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

				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).WithField("epoch", position.Next).WithContext(ctx).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

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

func (b *BeaconStatePendingConsolidationDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconStatePendingConsolidationDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))), //nolint:gosec // epoch fits int64
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))
	stateID := xatuethv1.SlotAsString(boundarySlot)

	consolidations, err := b.getPendingConsolidations(ctx, stateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch pending consolidations")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, len(consolidations))

	for index, consolidation := range consolidations {
		if consolidation == nil {
			continue
		}

		event, err := b.createEvent(ctx, consolidation, epoch, stateID, uint64(index))
		if err != nil {
			return nil, errors.Wrap(err, "failed to create event from pending consolidation")
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}

func (b *BeaconStatePendingConsolidationDeriver) getPendingConsolidations(ctx context.Context, stateID string) ([]*electra.PendingConsolidation, error) {
	provider, ok := b.beacon.Node().Service().(client.PendingConsolidationsProvider)
	if !ok {
		return nil, errors.New("pending consolidations provider not found")
	}

	rsp, err := provider.PendingConsolidations(ctx, &api.PendingConsolidationsOpts{
		State: stateID,
	})
	if err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (b *BeaconStatePendingConsolidationDeriver) createEvent(ctx context.Context, consolidation *electra.PendingConsolidation, epoch phase0.Epoch, stateID string, positionInQueue uint64) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	data := &xatu.PendingConsolidationData{
		SourceIndex: wrapperspb.UInt64(uint64(consolidation.SourceIndex)),
		TargetIndex: wrapperspb.UInt64(uint64(consolidation.TargetIndex)),
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconStatePendingConsolidation{
			EthV1BeaconStatePendingConsolidation: data,
		},
	}

	additionalData := b.getAdditionalData(ctx, epoch, stateID, positionInQueue)

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStatePendingConsolidation{
		EthV1BeaconStatePendingConsolidation: additionalData,
	}

	return decoratedEvent, nil
}

func (b *BeaconStatePendingConsolidationDeriver) getAdditionalData(_ context.Context, epoch phase0.Epoch, stateID string, positionInQueue uint64) *xatu.ClientMeta_AdditionalEthV1BeaconStatePendingConsolidationData {
	epochInfo := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1BeaconStatePendingConsolidationData{
		Epoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(uint64(epoch)),
			StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
		},
		StateId:         stateID,
		PositionInQueue: wrapperspb.UInt64(positionInQueue),
	}
}
