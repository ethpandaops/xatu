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

	v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BlockRewardDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOCK_REWARD
)

type BlockRewardDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"false"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type BlockRewardDeriver struct {
	log               observability.ContextualLogger
	cfg               *BlockRewardDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBlockRewardDeriver(log observability.ContextualLogger, config *BlockRewardDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BlockRewardDeriver {
	return &BlockRewardDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/block_reward",
			"type":   BlockRewardDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BlockRewardDeriver) CannonType() xatu.CannonType {
	return BlockRewardDeriverName
}

func (b *BlockRewardDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionAltair
}

func (b *BlockRewardDeriver) Name() string {
	return BlockRewardDeriverName.String()
}

func (b *BlockRewardDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BlockRewardDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Block reward deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Block reward deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *BlockRewardDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BlockRewardDeriver) run(rctx context.Context) {
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

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Location updated. Done.")

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
func (b *BlockRewardDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
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

func (b *BlockRewardDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BlockRewardDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))), //nolint:gosec // epoch fits int64
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := []*xatu.DecoratedEvent{}

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

func (b *BlockRewardDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BlockRewardDeriver.processSlot",
		trace.WithAttributes(attribute.Int64("slot", int64(slot))), //nolint:gosec // slot fits int64
	)
	defer span.End()

	// Get the block so we can confirm it exists and build the block identifier.
	block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	rewards, err := b.getBlockRewards(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block rewards for slot %d", slot)
	}

	if rewards == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	event, err := b.createEventFromBlockReward(ctx, rewards, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create event from block reward for slot %d", slot)
	}

	return []*xatu.DecoratedEvent{event}, nil
}

func (b *BlockRewardDeriver) getBlockRewards(ctx context.Context, blockID string) (*apiv1.BlockRewards, error) {
	provider, ok := b.beacon.Node().Service().(eth2client.BlockRewardsProvider)
	if !ok {
		return nil, errors.New("beacon node does not implement eth2client.BlockRewardsProvider")
	}

	response, err := provider.BlockRewards(ctx, &api.BlockRewardsOpts{
		Block: blockID,
	})
	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, errors.New("block rewards response is nil")
	}

	return response.Data, nil
}

func (b *BlockRewardDeriver) createEventFromBlockReward(ctx context.Context, rewards *apiv1.BlockRewards, block *spec.VersionedSignedBeaconBlock) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOCK_REWARD,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconBlockReward{
			EthV1BeaconBlockReward: &xatu.BlockRewardData{
				ProposerIndex:     wrapperspb.UInt64(uint64(rewards.ProposerIndex)),
				Total:             wrapperspb.UInt64(uint64(rewards.Total)),
				Attestations:      wrapperspb.UInt64(uint64(rewards.Attestations)),
				SyncAggregate:     wrapperspb.UInt64(uint64(rewards.SyncAggregate)),
				ProposerSlashings: wrapperspb.UInt64(uint64(rewards.ProposerSlashings)),
				AttesterSlashings: wrapperspb.UInt64(uint64(rewards.AttesterSlashings)),
			},
		},
	}

	additionalData, err := b.getAdditionalData(ctx, block)
	if err != nil {
		b.log.WithError(err).WithContext(ctx).Error("Failed to get extra block reward data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconBlockReward{
		EthV1BeaconBlockReward: additionalData,
	}

	return decoratedEvent, nil
}

func (b *BlockRewardDeriver) getAdditionalData(_ context.Context, block *spec.VersionedSignedBeaconBlock) (*xatu.ClientMeta_AdditionalEthV1BeaconBlockRewardData, error) {
	blockIdentifier, err := v2.GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block identifier")
	}

	return &xatu.ClientMeta_AdditionalEthV1BeaconBlockRewardData{
		Block: blockIdentifier,
	}, nil
}
