package v2

import (
	"context"
	"fmt"
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
	VoluntaryExitDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT
)

type VoluntaryExitDeriverConfig struct {
	Enabled     bool    `yaml:"enabled" default:"true"`
	HeadSlotLag *uint64 `yaml:"headSlotLag" default:"5"`
}

type VoluntaryExitDeriver struct {
	log                 logrus.FieldLogger
	cfg                 *VoluntaryExitDeriverConfig
	iterator            *iterator.SlotIterator
	onEventCallbacks    []func(ctx context.Context, event *xatu.DecoratedEvent) error
	onLocationCallbacks []func(ctx context.Context, location uint64) error
	beacon              *ethereum.BeaconNode
	clientMeta          *xatu.ClientMeta
}

func NewVoluntaryExitDeriver(log logrus.FieldLogger, config *VoluntaryExitDeriverConfig, iter *iterator.SlotIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *VoluntaryExitDeriver {
	return &VoluntaryExitDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/voluntary_exit"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *VoluntaryExitDeriver) CannonType() xatu.CannonType {
	return VoluntaryExitDeriverName
}

func (b *VoluntaryExitDeriver) Name() string {
	return VoluntaryExitDeriverName.String()
}

func (b *VoluntaryExitDeriver) OnEventDerived(ctx context.Context, fn func(ctx context.Context, event *xatu.DecoratedEvent) error) {
	b.onEventCallbacks = append(b.onEventCallbacks, fn)
}

func (b *VoluntaryExitDeriver) OnLocationUpdated(ctx context.Context, fn func(ctx context.Context, location uint64) error) {
	b.onLocationCallbacks = append(b.onLocationCallbacks, fn)
}

func (b *VoluntaryExitDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Voluntary exit deriver disabled")

		return nil
	}

	b.log.Info("Voluntary exit deriver enabled")

	// Start our main loop
	go b.run(ctx)

	return nil
}

func (b *VoluntaryExitDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *VoluntaryExitDeriver) run(ctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 1 * time.Minute

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
				location, err := b.iterator.Next(ctx)
				if err != nil {
					return err
				}

				for _, fn := range b.onLocationCallbacks {
					if errr := fn(ctx, location.GetEthV2BeaconBlockVoluntaryExit().GetSlot()); errr != nil {
						b.log.WithError(errr).Error("Failed to send location")
					}
				}

				// Process the slot
				events, err := b.processSlot(ctx, phase0.Slot(location.GetEthV2BeaconBlockVoluntaryExit().GetSlot()))
				if err != nil {
					b.log.WithError(err).Error("Failed to process slot")

					return err
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, location); err != nil {
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

func (b *VoluntaryExitDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
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

	exits, err := b.getVoluntaryExits(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, exit := range exits {
		event, err := b.createEvent(ctx, exit, blockIdentifier)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for voluntary exit %s", exit.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *VoluntaryExitDeriver) getVoluntaryExits(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.SignedVoluntaryExitV2, error) {
	exits := []*xatuethv1.SignedVoluntaryExitV2{}

	var voluntaryExits []*phase0.SignedVoluntaryExit

	switch block.Version {
	case spec.DataVersionPhase0:
		voluntaryExits = block.Phase0.Message.Body.VoluntaryExits
	case spec.DataVersionAltair:
		voluntaryExits = block.Altair.Message.Body.VoluntaryExits
	case spec.DataVersionBellatrix:
		voluntaryExits = block.Bellatrix.Message.Body.VoluntaryExits
	case spec.DataVersionCapella:
		voluntaryExits = block.Capella.Message.Body.VoluntaryExits
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version.String())
	}

	for _, exit := range voluntaryExits {
		exits = append(exits, &xatuethv1.SignedVoluntaryExitV2{
			Message: &xatuethv1.VoluntaryExitV2{
				Epoch:          wrapperspb.UInt64(uint64(exit.Message.Epoch)),
				ValidatorIndex: wrapperspb.UInt64(uint64(exit.Message.ValidatorIndex)),
			},
			Signature: exit.Signature.String(),
		})
	}

	return exits, nil
}

func (b *VoluntaryExitDeriver) createEvent(ctx context.Context, exit *xatuethv1.SignedVoluntaryExitV2, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: exit,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit{
		EthV2BeaconBlockVoluntaryExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockVoluntaryExitData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
