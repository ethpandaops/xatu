package v1

import (
	"context"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	eth2client "github.com/ethpandaops/go-eth2-client"
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
	AttestationRewardDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_ATTESTATION_REWARD
)

type AttestationRewardDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type AttestationRewardDeriver struct {
	log               observability.ContextualLogger
	cfg               *AttestationRewardDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewAttestationRewardDeriver(log observability.ContextualLogger, config *AttestationRewardDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *AttestationRewardDeriver {
	return &AttestationRewardDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/attestation_reward",
			"type":   AttestationRewardDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *AttestationRewardDeriver) CannonType() xatu.CannonType {
	return AttestationRewardDeriverName
}

func (b *AttestationRewardDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionAltair
}

func (b *AttestationRewardDeriver) Name() string {
	return AttestationRewardDeriverName.String()
}

func (b *AttestationRewardDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *AttestationRewardDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Attestation reward deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Attestation reward deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *AttestationRewardDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *AttestationRewardDeriver) run(rctx context.Context) {
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
					b.log.WithError(err).WithContext(ctx).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

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

func (b *AttestationRewardDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"AttestationRewardDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))), //nolint:gosec // epoch fits int64
	)
	defer span.End()

	provider, ok := b.beacon.Node().Service().(eth2client.AttestationRewardsProvider)
	if !ok {
		return nil, errors.New("beacon node does not implement AttestationRewardsProvider")
	}

	rsp, err := provider.AttestationRewards(ctx, &api.AttestationRewardsOpts{
		Epoch: epoch,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch attestation rewards")
	}

	if rsp == nil || rsp.Data == nil {
		return nil, errors.New("attestation rewards response is empty")
	}

	additionalData := b.getAdditionalData(ctx, epoch)

	allEvents := make([]*xatu.DecoratedEvent, 0, len(rsp.Data.TotalRewards))

	for _, reward := range rsp.Data.TotalRewards {
		event, err := b.createEventFromAttestationReward(reward, additionalData)
		if err != nil {
			b.log.
				WithError(err).
				WithField("validator_index", reward.ValidatorIndex).
				WithField("epoch", epoch).WithContext(ctx).
				Error("Failed to create event from attestation reward")

			return nil, err
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}

func (b *AttestationRewardDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	// Not supported.
}

func (b *AttestationRewardDeriver) createEventFromAttestationReward(reward apiv1.ValidatorAttestationRewards, additionalData *xatu.ClientMeta_AdditionalEthV1BeaconAttestationRewardData) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	var inclusionDelay *wrapperspb.UInt64Value
	if reward.InclusionDelay != nil {
		inclusionDelay = wrapperspb.UInt64(uint64(*reward.InclusionDelay))
	}

	metadata.AdditionalData = &xatu.ClientMeta_EthV1BeaconAttestationReward{
		EthV1BeaconAttestationReward: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_ATTESTATION_REWARD,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconAttestationReward{
			EthV1BeaconAttestationReward: &xatu.AttestationRewardData{
				ValidatorIndex: wrapperspb.UInt64(uint64(reward.ValidatorIndex)),
				Head:           wrapperspb.Int64(int64(reward.Head)), //nolint:gosec // reward value is bounded well within int64 range
				Target:         wrapperspb.Int64(reward.Target),
				Source:         wrapperspb.Int64(reward.Source),
				InclusionDelay: inclusionDelay,
				Inactivity:     wrapperspb.Int64(int64(reward.Inactivity)), //nolint:gosec // reward value is bounded well within int64 range
			},
		},
	}

	return decoratedEvent, nil
}

func (b *AttestationRewardDeriver) getAdditionalData(_ context.Context, epoch phase0.Epoch) *xatu.ClientMeta_AdditionalEthV1BeaconAttestationRewardData {
	epochWallclock := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1BeaconAttestationRewardData{
		StateId: xatuethv1.StateIDFinalized,
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: uint64(epoch)},
			StartDateTime: timestamppb.New(epochWallclock.TimeWindow().Start()),
		},
	}
}
