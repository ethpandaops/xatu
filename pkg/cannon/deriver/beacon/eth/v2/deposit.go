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
	DepositDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT
)

type DepositDeriverConfig struct {
	Enabled     bool    `yaml:"enabled" default:"true"`
	HeadSlotLag *uint64 `yaml:"headSlotLag" default:"1"`
}

type DepositDeriver struct {
	log                 logrus.FieldLogger
	cfg                 *DepositDeriverConfig
	iterator            *iterator.SlotIterator
	onEventCallbacks    []func(ctx context.Context, event *xatu.DecoratedEvent) error
	onLocationCallbacks []func(ctx context.Context, location uint64) error
	beacon              *ethereum.BeaconNode
	clientMeta          *xatu.ClientMeta
}

func NewDepositDeriver(log logrus.FieldLogger, config *DepositDeriverConfig, iter *iterator.SlotIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *DepositDeriver {
	return &DepositDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/deposit"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *DepositDeriver) CannonType() xatu.CannonType {
	return DepositDeriverName
}

func (b *DepositDeriver) Name() string {
	return DepositDeriverName.String()
}

func (b *DepositDeriver) OnEventDerived(ctx context.Context, fn func(ctx context.Context, event *xatu.DecoratedEvent) error) {
	b.onEventCallbacks = append(b.onEventCallbacks, fn)
}

func (b *DepositDeriver) OnLocationUpdated(ctx context.Context, fn func(ctx context.Context, location uint64) error) {
	b.onLocationCallbacks = append(b.onLocationCallbacks, fn)
}

func (b *DepositDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Deposit deriver disabled")

		return nil
	}

	b.log.Info("Deposit deriver enabled")

	// Start our main loop
	go b.run(ctx)

	return nil
}

func (b *DepositDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *DepositDeriver) run(ctx context.Context) {
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
					if errr := fn(ctx, location.GetEthV2BeaconBlockDeposit().GetSlot()); errr != nil {
						b.log.WithError(errr).Error("Failed to send location")
					}
				}

				// Process the slot
				events, err := b.processSlot(ctx, phase0.Slot(location.GetEthV2BeaconBlockDeposit().GetSlot()))
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

func (b *DepositDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
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

	deposits, err := b.getDeposits(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get deposits for block %s", blockIdentifier.String())
	}

	for _, deposit := range deposits {
		event, err := b.createEvent(ctx, deposit, blockIdentifier)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for deposit %s", deposit.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *DepositDeriver) getDeposits(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.DepositV2, error) {
	exits := []*xatuethv1.DepositV2{}

	var deposits []*phase0.Deposit

	switch block.Version {
	case spec.DataVersionPhase0:
		deposits = block.Phase0.Message.Body.Deposits
	case spec.DataVersionAltair:
		deposits = block.Altair.Message.Body.Deposits
	case spec.DataVersionBellatrix:
		deposits = block.Bellatrix.Message.Body.Deposits
	case spec.DataVersionCapella:
		deposits = block.Capella.Message.Body.Deposits
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version.String())
	}

	for _, deposit := range deposits {
		proof := []string{}
		for _, p := range deposit.Proof {
			proof = append(proof, fmt.Sprintf("0x%x", p))
		}

		exits = append(exits, &xatuethv1.DepositV2{
			Proof: proof,
			Data: &xatuethv1.DepositV2_Data{
				Pubkey:                deposit.Data.PublicKey.String(),
				WithdrawalCredentials: fmt.Sprintf("0x%x", deposit.Data.WithdrawalCredentials),
				Amount:                wrapperspb.UInt64(uint64(deposit.Data.Amount)),
				Signature:             deposit.Data.Signature.String(),
			},
		})
	}

	return exits, nil
}

func (b *DepositDeriver) createEvent(ctx context.Context, deposit *xatuethv1.DepositV2, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: deposit,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockDeposit{
		EthV2BeaconBlockDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockDepositData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
