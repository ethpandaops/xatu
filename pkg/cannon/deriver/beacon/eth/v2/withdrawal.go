package v2

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	WithdrawalDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL
)

type WithdrawalDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

type WithdrawalDeriver struct {
	log                 logrus.FieldLogger
	cfg                 *WithdrawalDeriverConfig
	iterator            *iterator.CheckpointIterator
	onEventsCallbacks   []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	onLocationCallbacks []func(ctx context.Context, location uint64) error
	beacon              *ethereum.BeaconNode
	clientMeta          *xatu.ClientMeta
}

func NewWithdrawalDeriver(log logrus.FieldLogger, config *WithdrawalDeriverConfig, iter *iterator.CheckpointIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *WithdrawalDeriver {
	return &WithdrawalDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/withdrawal"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *WithdrawalDeriver) CannonType() xatu.CannonType {
	return WithdrawalDeriverName
}

func (b *WithdrawalDeriver) Name() string {
	return WithdrawalDeriverName.String()
}

func (b *WithdrawalDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *WithdrawalDeriver) OnLocationUpdated(ctx context.Context, fn func(ctx context.Context, location uint64) error) {
	b.onLocationCallbacks = append(b.onLocationCallbacks, fn)
}

func (b *WithdrawalDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Withdrawal deriver disabled")

		return nil
	}

	b.log.Info("Withdrawal deriver enabled")

	// Start our main loop
	go b.run(ctx)

	return nil
}

func (b *WithdrawalDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *WithdrawalDeriver) run(ctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return
		default:
			operation := func() error {
				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					return err
				}

				// Get the next slot
				location, lookAhead, err := b.iterator.Next(ctx)
				if err != nil {
					return err
				}

				// Look ahead
				b.lookAheadAtLocation(ctx, lookAhead)

				for _, fn := range b.onLocationCallbacks {
					if errr := fn(ctx, location.GetEthV2BeaconBlockWithdrawal().GetEpoch()); errr != nil {
						b.log.WithError(errr).Error("Failed to send location")
					}
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, phase0.Epoch(location.GetEthV2BeaconBlockWithdrawal().GetEpoch()))
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					return err
				}

				for _, fn := range b.onEventsCallbacks {
					if errr := fn(ctx, events); errr != nil {
						return errors.Wrapf(errr, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, location); err != nil {
					return err
				}

				bo.Reset()

				return nil
			}

			if err := backoff.Retry(operation, bo); err != nil {
				b.log.WithError(err).Error("Failed to process epoch")
			}
		}
	}
}

func (b *WithdrawalDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
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

func (b *WithdrawalDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	// Get the block
	block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	events := []*xatu.DecoratedEvent{}

	withdrawals, err := b.getWithdrawals(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get withdrawals")
	}

	for _, withdrawal := range withdrawals {
		event, err := b.createEvent(ctx, withdrawal, blockIdentifier)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create event for withdrawal %s", withdrawal.String())
		}

		events = append(events, event)
	}

	return events, nil
}

// lookAheadAtLocation takes the upcoming locations and looks ahead to do any pre-processing that might be required.
func (b *WithdrawalDeriver) lookAheadAtLocation(ctx context.Context, locations []*xatu.CannonLocation) {
	if locations == nil {
		return
	}

	for _, location := range locations {
		// Get the next look ahead epoch
		epoch := phase0.Epoch(location.GetEthV2BeaconBlockWithdrawal().GetEpoch())

		sp, err := b.beacon.Node().Spec()
		if err != nil {
			b.log.WithError(err).WithField("epoch", epoch).Warn("Failed to look ahead at epoch")

			return
		}

		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			b.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (b *WithdrawalDeriver) getWithdrawals(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.WithdrawalV2, error) {
	withdrawals := []*xatuethv1.WithdrawalV2{}

	switch block.Version {
	case spec.DataVersionPhase0, spec.DataVersionAltair, spec.DataVersionBellatrix:
		return withdrawals, nil
	}

	for _, withdrawal := range block.Capella.Message.Body.ExecutionPayload.Withdrawals {
		withdrawals = append(withdrawals, &xatuethv1.WithdrawalV2{
			Index:          &wrapperspb.UInt64Value{Value: uint64(withdrawal.Index)},
			ValidatorIndex: &wrapperspb.UInt64Value{Value: uint64(withdrawal.ValidatorIndex)},
			Address:        withdrawal.Address.String(),
			Amount:         &wrapperspb.UInt64Value{Value: uint64(withdrawal.Amount)},
		})
	}

	return withdrawals, nil
}

func (b *WithdrawalDeriver) createEvent(ctx context.Context, withdrawal *xatuethv1.WithdrawalV2, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockWithdrawal{
			EthV2BeaconBlockWithdrawal: withdrawal,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockWithdrawal{
		EthV2BeaconBlockWithdrawal: &xatu.ClientMeta_AdditionalEthV2BeaconBlockWithdrawalData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
