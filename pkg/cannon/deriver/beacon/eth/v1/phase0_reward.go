package v1

import (
	"context"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	zcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	zphase0 "github.com/protolambda/zrnt/eth2/beacon/phase0"
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
	Phase0RewardDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_PHASE0_REWARD
)

type Phase0RewardDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"false"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// Phase0RewardDeriver computes the attestation and block rewards of phase0 epochs from beacon states, since the Beacon
// API rewards endpoints only serve Altair onwards. Epoch E's rows need the states at the last slots of epochs E-1, E and
// E+1; consecutive epochs share two of them, so each step downloads one new state.
type Phase0RewardDeriver struct {
	log               observability.ContextualLogger
	cfg               *Phase0RewardDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta

	// rewards caches computed epoch rewards by state epoch. Only the run loop touches it.
	rewards map[phase0.Epoch]*phase0EpochRewards
}

func NewPhase0RewardDeriver(log observability.ContextualLogger, config *Phase0RewardDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *Phase0RewardDeriver {
	return &Phase0RewardDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/phase0_reward",
			"type":   Phase0RewardDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
		rewards:    map[phase0.Epoch]*phase0EpochRewards{},
	}
}

func (b *Phase0RewardDeriver) CannonType() xatu.CannonType {
	return Phase0RewardDeriverName
}

func (b *Phase0RewardDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (b *Phase0RewardDeriver) Name() string {
	return Phase0RewardDeriverName.String()
}

func (b *Phase0RewardDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *Phase0RewardDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Phase0 reward deriver disabled")

		return nil
	}

	altair, err := b.altairEpoch()
	if err != nil {
		return err
	}

	if altair == 0 {
		b.log.WithContext(ctx).Info("Phase0 reward deriver idle: the network starts at Altair")

		return nil
	}

	b.log.WithContext(ctx).WithField("altair_epoch", altair).Info("Phase0 reward deriver enabled")

	if err := b.iterator.StartUntil(ctx, b.ActivationFork(), spec.DataVersionAltair); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx)

	return nil
}

func (b *Phase0RewardDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *Phase0RewardDeriver) run(rctx context.Context) {
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

				position, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				b.lookAhead(ctx, position.LookAheads)

				events, err := iterator.DeriveEpochs(ctx, position.Epochs(), b.processEpoch)
				if err != nil {
					b.log.WithError(err).WithContext(ctx).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				if err := b.iterator.UpdateLocation(ctx, position.LastEpoch(), position.Direction); err != nil {
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

// lookAhead preloads the blocks of the upcoming epochs.
func (b *Phase0RewardDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).WithContext(ctx).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i < uint64(sp.SlotsPerEpoch); i++ {
			b.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(phase0.Slot(uint64(epoch)*uint64(sp.SlotsPerEpoch) + i)))
		}
	}
}

func (b *Phase0RewardDeriver) altairEpoch() (phase0.Epoch, error) {
	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return 0, errors.Wrap(err, "failed to obtain spec")
	}

	fork, err := sp.ForkEpochs.GetByName(spec.DataVersionAltair.String())
	if err != nil {
		return 0, errors.Wrap(err, "failed to get the altair fork epoch")
	}

	return fork.Epoch, nil
}

func (b *Phase0RewardDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"Phase0RewardDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))), //nolint:gosec // epoch fits int64
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	slotsPerEpoch := uint64(sp.SlotsPerEpoch)

	altair, err := b.altairEpoch()
	if err != nil {
		return nil, err
	}

	if epoch >= altair {
		return nil, fmt.Errorf("epoch %d is not before altair (epoch %d)", epoch, altair)
	}

	var previous, next *phase0EpochRewards

	if epoch > 0 {
		if previous, err = b.epochRewards(ctx, epoch-1, slotsPerEpoch); err != nil {
			return nil, err
		}
	}

	current, err := b.epochRewards(ctx, epoch, slotsPerEpoch)
	if err != nil {
		return nil, err
	}

	// The transition ending the last phase0 epoch runs the phase0 rules, but upgrades the state to Altair: the
	// attestations of that epoch are rewarded by Altair's rules, so it has no phase0 attestation rows.
	if epoch+1 < altair {
		if next, err = b.epochRewards(ctx, epoch+1, slotsPerEpoch); err != nil {
			return nil, err
		}
	}

	// Keep what the neighbouring epochs reuse, whichever direction the iterator walks.
	for cached := range b.rewards {
		if cached+1 < epoch || cached > epoch+1 {
			delete(b.rewards, cached)
		}
	}

	blocks, versioned, err := b.epochBlocks(ctx, epoch, slotsPerEpoch)
	if err != nil {
		return nil, err
	}

	blockRewards, err := computePhase0BlockRewards(blocks, slotsPerEpoch, previous, current, next)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute block rewards for epoch %d", epoch)
	}

	events := make([]*xatu.DecoratedEvent, 0, len(blockRewards))

	for i := range blockRewards {
		event, err := b.createBlockRewardEvent(&blockRewards[i], versioned[i])
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if next == nil {
		return events, nil
	}

	additionalData := b.attestationRewardAdditionalData(epoch, slotsPerEpoch)

	for i := range next.Attesters {
		event, err := b.createAttestationRewardEvent(&next.Attesters[i], additionalData)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

// epochRewards returns the rewards read from the state at the last slot of stateEpoch.
func (b *Phase0RewardDeriver) epochRewards(ctx context.Context, stateEpoch phase0.Epoch, slotsPerEpoch uint64) (*phase0EpochRewards, error) {
	if rewards, ok := b.rewards[stateEpoch]; ok {
		return rewards, nil
	}

	slot := phase0.Slot((uint64(stateEpoch)+1)*slotsPerEpoch - 1)

	var rewards *phase0EpochRewards

	err := b.beacon.WithPhase0BeaconState(ctx, xatuethv1.SlotAsString(slot), func(zspec *zcommon.Spec, state *zphase0.BeaconStateView) error {
		stateSlot, err := state.Slot()
		if err != nil {
			return err
		}

		if uint64(stateSlot) != uint64(slot) {
			return fmt.Errorf("requested the state at slot %d, got slot %d", slot, stateSlot)
		}

		rewards, err = computePhase0EpochRewards(ctx, zspec, state)

		return err
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute rewards from the state at slot %d", slot)
	}

	b.rewards[stateEpoch] = rewards

	return rewards, nil
}

// epochBlocks returns the canonical blocks of the epoch in slot order.
func (b *Phase0RewardDeriver) epochBlocks(ctx context.Context, epoch phase0.Epoch, slotsPerEpoch uint64) ([]*phase0.BeaconBlock, []*spec.VersionedSignedBeaconBlock, error) {
	blocks := make([]*phase0.BeaconBlock, 0, slotsPerEpoch)
	versioned := make([]*spec.VersionedSignedBeaconBlock, 0, slotsPerEpoch)

	for i := uint64(0); i < slotsPerEpoch; i++ {
		slot := phase0.Slot(uint64(epoch)*slotsPerEpoch + i)

		block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
		}

		if block == nil {
			continue
		}

		if block.Version != spec.DataVersionPhase0 || block.Phase0 == nil || block.Phase0.Message == nil {
			return nil, nil, fmt.Errorf("block at slot %d is %s, expected phase0", slot, block.Version)
		}

		blocks = append(blocks, block.Phase0.Message)
		versioned = append(versioned, block)
	}

	return blocks, versioned, nil
}

func (b *Phase0RewardDeriver) createBlockRewardEvent(reward *phase0BlockReward, block *spec.VersionedSignedBeaconBlock) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	blockIdentifier, err := v2.GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block identifier")
	}

	metadata.AdditionalData = &xatu.ClientMeta_EthV1BeaconPhase0BlockReward{
		EthV1BeaconPhase0BlockReward: &xatu.ClientMeta_AdditionalEthV1BeaconBlockRewardData{
			Block: blockIdentifier,
		},
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_PHASE0_BLOCK_REWARD,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconPhase0BlockReward{
			EthV1BeaconPhase0BlockReward: &xatu.BlockRewardData{
				ProposerIndex:     wrapperspb.UInt64(uint64(reward.ProposerIndex)),
				Total:             wrapperspb.UInt64(reward.Total()),
				Attestations:      wrapperspb.UInt64(reward.Attestations),
				ProposerSlashings: wrapperspb.UInt64(reward.ProposerSlashings),
				AttesterSlashings: wrapperspb.UInt64(reward.AttesterSlashings),
			},
		},
	}, nil
}

func (b *Phase0RewardDeriver) createAttestationRewardEvent(reward *phase0AttesterReward, additionalData *xatu.ClientMeta_AdditionalEthV1BeaconAttestationRewardData) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_EthV1BeaconPhase0AttestationReward{
		EthV1BeaconPhase0AttestationReward: additionalData,
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_PHASE0_ATTESTATION_REWARD,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconPhase0AttestationReward{
			EthV1BeaconPhase0AttestationReward: &xatu.AttestationRewardData{
				ValidatorIndex: wrapperspb.UInt64(uint64(reward.ValidatorIndex)),
				Head:           wrapperspb.Int64(reward.Head),
				Target:         wrapperspb.Int64(reward.Target),
				Source:         wrapperspb.Int64(reward.Source),
				InclusionDelay: wrapperspb.UInt64(reward.InclusionDelay),
				Inactivity:     wrapperspb.Int64(reward.Inactivity),
			},
		},
	}, nil
}

// attestationRewardAdditionalData describes epoch's rewards, read from the state at the last slot of the next epoch.
func (b *Phase0RewardDeriver) attestationRewardAdditionalData(epoch phase0.Epoch, slotsPerEpoch uint64) *xatu.ClientMeta_AdditionalEthV1BeaconAttestationRewardData {
	epochWallclock := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1BeaconAttestationRewardData{
		StateId: xatuethv1.SlotAsString(phase0.Slot((uint64(epoch)+2)*slotsPerEpoch - 1)),
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: uint64(epoch)},
			StartDateTime: timestamppb.New(epochWallclock.TimeWindow().Start()),
		},
	}
}
