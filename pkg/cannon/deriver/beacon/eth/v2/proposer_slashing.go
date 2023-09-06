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
	ProposerSlashingDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING
)

type ProposerSlashingDeriverConfig struct {
	Enabled     bool    `yaml:"enabled" default:"true"`
	HeadSlotLag *uint64 `yaml:"headSlotLag" default:"1"`
}

type ProposerSlashingDeriver struct {
	log                 logrus.FieldLogger
	cfg                 *ProposerSlashingDeriverConfig
	iterator            *iterator.SlotIterator
	onEventCallbacks    []func(ctx context.Context, event *xatu.DecoratedEvent) error
	onLocationCallbacks []func(ctx context.Context, location uint64) error
	beacon              *ethereum.BeaconNode
	clientMeta          *xatu.ClientMeta
}

func NewProposerSlashingDeriver(log logrus.FieldLogger, config *ProposerSlashingDeriverConfig, iter *iterator.SlotIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ProposerSlashingDeriver {
	return &ProposerSlashingDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/proposer_slashing"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *ProposerSlashingDeriver) CannonType() xatu.CannonType {
	return ProposerSlashingDeriverName
}

func (b *ProposerSlashingDeriver) Name() string {
	return ProposerSlashingDeriverName.String()
}

func (b *ProposerSlashingDeriver) OnEventDerived(ctx context.Context, fn func(ctx context.Context, event *xatu.DecoratedEvent) error) {
	b.onEventCallbacks = append(b.onEventCallbacks, fn)
}

func (b *ProposerSlashingDeriver) OnLocationUpdated(ctx context.Context, fn func(ctx context.Context, location uint64) error) {
	b.onLocationCallbacks = append(b.onLocationCallbacks, fn)
}

func (b *ProposerSlashingDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Proposer slashing deriver disabled")

		return nil
	}

	b.log.Info("Proposer slashing deriver enabled")

	// Start our main loop
	go b.run(ctx)

	return nil
}

func (b *ProposerSlashingDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *ProposerSlashingDeriver) run(ctx context.Context) {
	bo := backoff.NewExponentialBackOff()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			operation := func() error {
				time.Sleep(100 * time.Millisecond)

				// Get the next slot
				location, err := b.iterator.Next(ctx)
				if err != nil {
					return err
				}

				for _, fn := range b.onLocationCallbacks {
					if errr := fn(ctx, location.GetEthV2BeaconBlockProposerSlashing().GetSlot()); errr != nil {
						b.log.WithError(errr).Error("Failed to send location")
					}
				}

				// Process the slot
				events, err := b.processSlot(ctx, phase0.Slot(location.GetEthV2BeaconBlockProposerSlashing().GetSlot()))
				if err != nil {
					b.log.WithError(err).Error("Failed to process slot")

					return err
				}

				// Send the events
				for _, event := range events {
					for _, fn := range b.onEventCallbacks {
						if err := fn(ctx, event); err != nil {
							b.log.WithError(err).Error("Failed to send event")
						}
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, location); err != nil {
					return err
				}

				bo.Reset()

				return nil
			}

			if err := backoff.RetryNotify(operation, bo, func(err error, timer time.Duration) {
				b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
			}); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

func (b *ProposerSlashingDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
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

	slashings, err := b.getProposerSlashings(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, slashing := range slashings {
		event, err := b.createEvent(ctx, slashing, blockIdentifier)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for proposer slashing %s", slashing.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *ProposerSlashingDeriver) getProposerSlashings(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.ProposerSlashingV2, error) {
	slashings := []*xatuethv1.ProposerSlashingV2{}

	blockSlashings, err := block.ProposerSlashings()
	if err != nil {
		return nil, err
	}

	for _, slashing := range blockSlashings {
		slashings = append(slashings, &xatuethv1.ProposerSlashingV2{
			SignedHeader_1: &xatuethv1.SignedBeaconBlockHeaderV2{
				Message: &xatuethv1.BeaconBlockHeaderV2{
					Slot:          wrapperspb.UInt64(uint64(slashing.SignedHeader1.Message.Slot)),
					ProposerIndex: wrapperspb.UInt64(uint64(slashing.SignedHeader1.Message.ProposerIndex)),
					ParentRoot:    slashing.SignedHeader1.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader1.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader1.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader1.Signature.String(),
			},
			SignedHeader_2: &xatuethv1.SignedBeaconBlockHeaderV2{
				Message: &xatuethv1.BeaconBlockHeaderV2{
					Slot:          wrapperspb.UInt64(uint64(slashing.SignedHeader2.Message.Slot)),
					ProposerIndex: wrapperspb.UInt64(uint64(slashing.SignedHeader2.Message.ProposerIndex)),
					ParentRoot:    slashing.SignedHeader2.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader2.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader2.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader2.Signature.String(),
			},
		})
	}

	return slashings, nil
}

func (b *ProposerSlashingDeriver) createEvent(ctx context.Context, slashing *xatuethv1.ProposerSlashingV2, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: slashing,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockProposerSlashing{
		EthV2BeaconBlockProposerSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockProposerSlashingData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
