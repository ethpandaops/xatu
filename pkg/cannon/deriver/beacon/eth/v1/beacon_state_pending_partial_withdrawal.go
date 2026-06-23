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
	BeaconStatePendingPartialWithdrawalDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL
)

// BeaconStatePendingPartialWithdrawalDeriverConfig configures the pending
// partial withdrawal queue deriver.
type BeaconStatePendingPartialWithdrawalDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// BeaconStatePendingPartialWithdrawalDeriver derives the Electra pending partial
// withdrawal queue for each epoch boundary.
type BeaconStatePendingPartialWithdrawalDeriver struct {
	log               observability.ContextualLogger
	cfg               *BeaconStatePendingPartialWithdrawalDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

// NewBeaconStatePendingPartialWithdrawalDeriver creates a new deriver.
func NewBeaconStatePendingPartialWithdrawalDeriver(log observability.ContextualLogger, config *BeaconStatePendingPartialWithdrawalDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconStatePendingPartialWithdrawalDeriver {
	return &BeaconStatePendingPartialWithdrawalDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/beacon_state_pending_partial_withdrawal",
			"type":   BeaconStatePendingPartialWithdrawalDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) CannonType() xatu.CannonType {
	return BeaconStatePendingPartialWithdrawalDeriverName
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionElectra
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) Name() string {
	return BeaconStatePendingPartialWithdrawalDeriverName.String()
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) Start(ctx context.Context) error {
	b.log.WithFields(logrus.Fields{
		"enabled": b.cfg.Enabled,
	}).WithContext(ctx).Info("Starting BeaconStatePendingPartialWithdrawalDeriver")

	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Pending partial withdrawal deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Pending partial withdrawal deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx)

	return nil
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) run(rctx context.Context) {
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

func (b *BeaconStatePendingPartialWithdrawalDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconStatePendingPartialWithdrawalDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))
	stateID := xatuethv1.SlotAsString(boundarySlot)

	withdrawals, err := b.getPendingPartialWithdrawals(ctx, stateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch pending partial withdrawals")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, len(withdrawals))

	for position, withdrawal := range withdrawals {
		if withdrawal == nil {
			continue
		}

		event, err := b.createEvent(ctx, withdrawal, uint64(position), epoch, stateID)
		if err != nil {
			b.log.
				WithError(err).
				WithField("position", position).
				WithField("epoch", epoch).WithContext(ctx).
				Error("Failed to create event from pending partial withdrawal")

			return nil, err
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) getPendingPartialWithdrawals(ctx context.Context, stateID string) ([]*electra.PendingPartialWithdrawal, error) {
	provider, isProvider := b.beacon.Node().Service().(client.PendingPartialWithdrawalsProvider)
	if !isProvider {
		return nil, errors.New("pending partial withdrawals client not found")
	}

	response, err := provider.PendingPartialWithdrawals(ctx, &api.PendingPartialWithdrawalsOpts{
		State: stateID,
		Common: api.CommonOpts{
			Timeout: 6 * time.Minute,
		},
	})
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) createEvent(ctx context.Context, withdrawal *electra.PendingPartialWithdrawal, position uint64, epoch phase0.Epoch, stateID string) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	data := &xatu.PendingPartialWithdrawalData{
		ValidatorIndex:    wrapperspb.UInt64(uint64(withdrawal.ValidatorIndex)),
		Amount:            wrapperspb.UInt64(uint64(withdrawal.Amount)),
		WithdrawableEpoch: wrapperspb.UInt64(uint64(withdrawal.WithdrawableEpoch)),
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconStatePendingPartialWithdrawal{
			EthV1BeaconStatePendingPartialWithdrawal: data,
		},
	}

	additionalData, err := b.getAdditionalData(ctx, position, epoch, stateID)
	if err != nil {
		b.log.WithError(err).WithContext(ctx).Error("Failed to get extra pending partial withdrawal data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStatePendingPartialWithdrawal{
		EthV1BeaconStatePendingPartialWithdrawal: additionalData,
	}

	return decoratedEvent, nil
}

func (b *BeaconStatePendingPartialWithdrawalDeriver) getAdditionalData(_ context.Context, position uint64, epoch phase0.Epoch, stateID string) (*xatu.ClientMeta_AdditionalEthV1BeaconStatePendingPartialWithdrawalData, error) {
	epochInfo := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1BeaconStatePendingPartialWithdrawalData{
		Epoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(uint64(epoch)),
			StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
		},
		StateId:         stateID,
		PositionInQueue: wrapperspb.UInt64(position),
	}, nil
}
