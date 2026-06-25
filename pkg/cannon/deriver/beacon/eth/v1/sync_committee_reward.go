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

	v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	SyncCommitteeRewardDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD
)

type SyncCommitteeRewardDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type SyncCommitteeRewardDeriver struct {
	log               observability.ContextualLogger
	cfg               *SyncCommitteeRewardDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewSyncCommitteeRewardDeriver(
	log observability.ContextualLogger,
	config *SyncCommitteeRewardDeriverConfig,
	iter *iterator.BackfillingCheckpoint,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *SyncCommitteeRewardDeriver {
	return &SyncCommitteeRewardDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/sync_committee_reward",
			"type":   SyncCommitteeRewardDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *SyncCommitteeRewardDeriver) CannonType() xatu.CannonType {
	return SyncCommitteeRewardDeriverName
}

func (b *SyncCommitteeRewardDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionAltair
}

func (b *SyncCommitteeRewardDeriver) Name() string {
	return SyncCommitteeRewardDeriverName.String()
}

func (b *SyncCommitteeRewardDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *SyncCommitteeRewardDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Sync committee reward deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Sync committee reward deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx)

	return nil
}

func (b *SyncCommitteeRewardDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *SyncCommitteeRewardDeriver) run(rctx context.Context) {
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

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).WithContext(ctx).Error("Failed to process epoch")

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

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (b *SyncCommitteeRewardDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).WithContext(ctx).Warn("Failed to look ahead at epoch")

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

func (b *SyncCommitteeRewardDeriver) processEpoch(
	ctx context.Context,
	epoch phase0.Epoch,
) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"SyncCommitteeRewardDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))), //nolint:gosec // epoch fits int64
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, sp.SlotsPerEpoch)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := b.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (b *SyncCommitteeRewardDeriver) processSlot(
	ctx context.Context,
	slot phase0.Slot,
) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"SyncCommitteeRewardDeriver.processSlot",
		//nolint:gosec // slot value will never overflow int64
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block. Sync committee rewards are only produced for slots that
	// contain a canonical block.
	block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	// Sync committee rewards were introduced in Altair.
	if block.Version < spec.DataVersionAltair {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := v2.GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block identifier")
	}

	rewards, err := b.fetchSyncCommitteeRewards(ctx, blockIdentifier.GetRoot())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch sync committee rewards for slot %d", slot)
	}

	events := make([]*xatu.DecoratedEvent, 0, len(rewards))

	for _, reward := range rewards {
		event, err := b.createEvent(ctx, reward, blockIdentifier)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create event for validator %d", reward.ValidatorIndex)
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *SyncCommitteeRewardDeriver) fetchSyncCommitteeRewards(
	ctx context.Context,
	blockID string,
) ([]*apiv1.SyncCommitteeReward, error) {
	provider, isProvider := b.beacon.Node().Service().(client.SyncCommitteeRewardsProvider)
	if !isProvider {
		return nil, errors.New("beacon node does not support sync committee rewards provider")
	}

	rsp, err := provider.SyncCommitteeRewards(ctx, &api.SyncCommitteeRewardsOpts{
		Block: blockID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sync committee rewards from beacon node")
	}

	if rsp == nil || rsp.Data == nil {
		return nil, errors.New("sync committee rewards response is nil")
	}

	return rsp.Data, nil
}

func (b *SyncCommitteeRewardDeriver) createEvent(
	ctx context.Context,
	reward *apiv1.SyncCommitteeReward,
	blockIdentifier *xatu.BlockIdentifier,
) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconSyncCommitteeReward{
			EthV1BeaconSyncCommitteeReward: &xatu.SyncCommitteeRewardData{
				ValidatorIndex: wrapperspb.UInt64(uint64(reward.ValidatorIndex)),
				Reward:         wrapperspb.Int64(reward.Reward),
			},
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconSyncCommitteeReward{
		EthV1BeaconSyncCommitteeReward: &xatu.ClientMeta_AdditionalEthV1BeaconSyncCommitteeRewardData{
			Block: blockIdentifier,
		},
	}

	return decoratedEvent, nil
}
