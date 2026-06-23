package v1

import (
	"context"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
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

	client "github.com/ethpandaops/go-eth2-client"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconStatePendingDepositDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT
)

type BeaconStatePendingDepositDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type BeaconStatePendingDepositDeriver struct {
	log               observability.ContextualLogger
	cfg               *BeaconStatePendingDepositDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBeaconStatePendingDepositDeriver(log observability.ContextualLogger, config *BeaconStatePendingDepositDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconStatePendingDepositDeriver {
	return &BeaconStatePendingDepositDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/beacon_state_pending_deposit",
			"type":   BeaconStatePendingDepositDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconStatePendingDepositDeriver) CannonType() xatu.CannonType {
	return BeaconStatePendingDepositDeriverName
}

func (b *BeaconStatePendingDepositDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionElectra
}

func (b *BeaconStatePendingDepositDeriver) Name() string {
	return BeaconStatePendingDepositDeriverName.String()
}

func (b *BeaconStatePendingDepositDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconStatePendingDepositDeriver) Start(ctx context.Context) error {
	b.log.WithField("enabled", b.cfg.Enabled).WithContext(ctx).Info("Starting BeaconStatePendingDepositDeriver")

	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Pending deposit deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Pending deposit deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx)

	return nil
}

func (b *BeaconStatePendingDepositDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconStatePendingDepositDeriver) run(rctx context.Context) {
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

func (b *BeaconStatePendingDepositDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconStatePendingDepositDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))), //nolint:gosec // epoch fits int64
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))
	stateID := xatuethv1.SlotAsString(boundarySlot)

	deposits, err := b.getPendingDeposits(ctx, stateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch pending deposits")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, len(deposits))

	for index, deposit := range deposits {
		if deposit == nil {
			continue
		}

		event, err := b.createEvent(ctx, deposit, uint64(index), stateID, epoch)
		if err != nil {
			b.log.
				WithError(err).
				WithField("position_in_queue", index).
				WithField("epoch", epoch).WithContext(ctx).
				Error("Failed to create event from pending deposit")

			return nil, err
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}

func (b *BeaconStatePendingDepositDeriver) getPendingDeposits(ctx context.Context, stateID string) ([]*electra.PendingDeposit, error) {
	provider, isProvider := b.beacon.Node().Service().(client.PendingDepositProvider)
	if !isProvider {
		return nil, errors.New("pending deposit provider not found")
	}

	response, err := provider.PendingDeposits(ctx, &api.PendingDepositsOpts{
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

func (b *BeaconStatePendingDepositDeriver) createEvent(ctx context.Context, deposit *electra.PendingDeposit, positionInQueue uint64, stateID string, epoch phase0.Epoch) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	data := &xatu.PendingDepositData{
		Pubkey:                deposit.Pubkey.String(),
		WithdrawalCredentials: fmt.Sprintf("%#x", deposit.WithdrawalCredentials),
		Amount:                wrapperspb.UInt64(uint64(deposit.Amount)),
		Signature:             deposit.Signature.String(),
		Slot:                  wrapperspb.UInt64(uint64(deposit.Slot)),
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconStatePendingDeposit{
			EthV1BeaconStatePendingDeposit: data,
		},
	}

	additionalData := b.getAdditionalData(ctx, positionInQueue, stateID, epoch)

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStatePendingDeposit{
		EthV1BeaconStatePendingDeposit: additionalData,
	}

	return decoratedEvent, nil
}

func (b *BeaconStatePendingDepositDeriver) getAdditionalData(_ context.Context, positionInQueue uint64, stateID string, epoch phase0.Epoch) *xatu.ClientMeta_AdditionalEthV1BeaconStatePendingDepositData {
	epochInfo := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1BeaconStatePendingDepositData{
		Epoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(uint64(epoch)),
			StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
		},
		StateId:         stateID,
		PositionInQueue: wrapperspb.UInt64(positionInQueue),
	}
}
