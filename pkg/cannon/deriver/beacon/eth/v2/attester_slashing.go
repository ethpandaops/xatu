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
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	AttesterSlashingDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING
)

type AttesterSlashingDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

type AttesterSlashingDeriver struct {
	log               logrus.FieldLogger
	cfg               *AttesterSlashingDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewAttesterSlashingDeriver(log logrus.FieldLogger, config *AttesterSlashingDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *AttesterSlashingDeriver {
	return &AttesterSlashingDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/attester_slashing"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (a *AttesterSlashingDeriver) CannonType() xatu.CannonType {
	return AttesterSlashingDeriverName
}

func (a *AttesterSlashingDeriver) ActivationFork() string {
	return ethereum.ForkNamePhase0
}

func (a *AttesterSlashingDeriver) Name() string {
	return AttesterSlashingDeriverName.String()
}

func (a *AttesterSlashingDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	a.onEventsCallbacks = append(a.onEventsCallbacks, fn)
}

func (a *AttesterSlashingDeriver) Start(ctx context.Context) error {
	if !a.cfg.Enabled {
		a.log.Info("Attester slashing deriver disabled")

		return nil
	}

	a.log.Info("Attester slashing deriver enabled")

	if err := a.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	go a.run(ctx)

	return nil
}

func (a *AttesterSlashingDeriver) Stop(ctx context.Context) error {
	return nil
}

func (a *AttesterSlashingDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() error {
				ctx, span := observability.Tracer().Start(rctx,
					fmt.Sprintf("Derive %s", a.Name()),
					trace.WithAttributes(
						attribute.String("network", string(a.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := a.beacon.Synced(ctx); err != nil {
					return err
				}

				// Get the next slot
				position, err := a.iterator.Next(ctx)
				if err != nil {
					return err
				}

				// Process the epoch
				events, err := a.processEpoch(ctx, position.Next)
				if err != nil {
					a.log.WithError(err).Error("Failed to process epoch")

					return err
				}

				// Look ahead
				a.lookAhead(ctx, position.LookAheads)

				// Send the events
				for _, fn := range a.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						return errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := a.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					return err
				}

				bo.Reset()

				return nil
			}

			if err := backoff.RetryNotify(operation, bo, func(err error, timer time.Duration) {
				a.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
			}); err != nil {
				a.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (a *AttesterSlashingDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"AttesterSlashingDeriver.lookAhead",
	)
	defer span.End()

	sp, err := a.beacon.Node().Spec()
	if err != nil {
		a.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			a.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (a *AttesterSlashingDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"AttesterSlashingDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := a.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := []*xatu.DecoratedEvent{}

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := a.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (a *AttesterSlashingDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"AttesterSlashingDeriver.processSlot",
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := a.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, a.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	events := []*xatu.DecoratedEvent{}

	slashings, err := a.getAttesterSlashings(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get attester slashings for slot %d", slot)
	}

	for _, slashing := range slashings {
		event, err := a.createEvent(ctx, slashing, blockIdentifier)
		if err != nil {
			a.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for attester slashing %s", slashing.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (a *AttesterSlashingDeriver) getAttesterSlashings(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.AttesterSlashingV2, error) {
	slashings := []*xatuethv1.AttesterSlashingV2{}

	attesterSlashings, err := block.AttesterSlashings()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attester slashings")
	}

	for _, slashing := range attesterSlashings {
		att1, err := slashing.Attestation1()
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain attestation 1")
		}

		indexedAttestation1, err := convertIndexedAttestation(att1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert indexed attestation 1")
		}

		att2, err := slashing.Attestation2()
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain attestation 2")
		}

		indexedAttestation2, err := convertIndexedAttestation(att2)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert indexed attestation 2")
		}

		slashings = append(slashings, &xatuethv1.AttesterSlashingV2{
			Attestation_1: indexedAttestation1,
			Attestation_2: indexedAttestation2,
		})
	}

	return slashings, nil
}

func convertIndexedAttestation(attestation *spec.VersionedIndexedAttestation) (*xatuethv1.IndexedAttestationV2, error) {
	indicies := []*wrapperspb.UInt64Value{}

	atIndicies, err := attestation.AttestingIndices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attesting indices")
	}

	for _, index := range atIndicies {
		indicies = append(indicies, &wrapperspb.UInt64Value{Value: index})
	}

	data, err := attestation.Data()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation data")
	}

	sig, err := attestation.Signature()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation signature")
	}

	return &xatuethv1.IndexedAttestationV2{
		AttestingIndices: indicies,
		Data: &xatuethv1.AttestationDataV2{
			Slot:            &wrapperspb.UInt64Value{Value: uint64(data.Slot)},
			Index:           &wrapperspb.UInt64Value{Value: uint64(data.Index)},
			BeaconBlockRoot: data.BeaconBlockRoot.String(),
			Source: &xatuethv1.CheckpointV2{
				Epoch: &wrapperspb.UInt64Value{Value: uint64(data.Source.Epoch)},
				Root:  data.Source.Root.String(),
			},
			Target: &xatuethv1.CheckpointV2{
				Epoch: &wrapperspb.UInt64Value{Value: uint64(data.Target.Epoch)},
				Root:  data.Target.Root.String(),
			},
		},
		Signature: sig.String(),
	}, nil
}

func (a *AttesterSlashingDeriver) createEvent(ctx context.Context, slashing *xatuethv1.AttesterSlashingV2, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(a.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: slashing,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockAttesterSlashing{
		EthV2BeaconBlockAttesterSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAttesterSlashingData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
