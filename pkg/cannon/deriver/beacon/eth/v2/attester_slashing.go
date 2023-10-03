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
	log                 logrus.FieldLogger
	cfg                 *AttesterSlashingDeriverConfig
	iterator            *iterator.CheckpointIterator
	onEventsCallbacks   []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	onLocationCallbacks []func(ctx context.Context, loc uint64) error
	beacon              *ethereum.BeaconNode
	clientMeta          *xatu.ClientMeta
}

func NewAttesterSlashingDeriver(log logrus.FieldLogger, config *AttesterSlashingDeriverConfig, iter *iterator.CheckpointIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *AttesterSlashingDeriver {
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

func (a *AttesterSlashingDeriver) Name() string {
	return AttesterSlashingDeriverName.String()
}

func (a *AttesterSlashingDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	a.onEventsCallbacks = append(a.onEventsCallbacks, fn)
}

func (a *AttesterSlashingDeriver) OnLocationUpdated(ctx context.Context, fn func(ctx context.Context, loc uint64) error) {
	a.onLocationCallbacks = append(a.onLocationCallbacks, fn)
}

func (a *AttesterSlashingDeriver) Start(ctx context.Context) error {
	if !a.cfg.Enabled {
		a.log.Info("Attester slashing deriver disabled")

		return nil
	}

	a.log.Info("Attester slashing deriver enabled")

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
				ctx, span := observability.Tracer().Start(rctx, fmt.Sprintf("Derive %s", a.Name()))
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := a.beacon.Synced(ctx); err != nil {
					return err
				}

				// Get the next slot
				location, lookAhead, err := a.iterator.Next(ctx)
				if err != nil {
					return err
				}

				// Look ahead
				a.lookAheadAtLocations(ctx, lookAhead)

				for _, fn := range a.onLocationCallbacks {
					if errr := fn(ctx, location.GetEthV2BeaconBlockAttesterSlashing().GetEpoch()); errr != nil {
						a.log.WithError(errr).Error("Failed to send location")
					}
				}

				// Process the epoch
				events, err := a.processEpoch(ctx, phase0.Epoch(location.GetEthV2BeaconBlockAttesterSlashing().GetEpoch()))
				if err != nil {
					a.log.WithError(err).Error("Failed to process epoch")

					return err
				}

				// Send the events
				for _, fn := range a.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						return errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := a.iterator.UpdateLocation(ctx, location); err != nil {
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

// lookAheadAtLocation takes the upcoming locations and looks ahead to do any pre-processing that might be required.
func (a *AttesterSlashingDeriver) lookAheadAtLocations(ctx context.Context, locations []*xatu.CannonLocation) {
	_, span := observability.Tracer().Start(ctx,
		"AttesterSlashingDeriver.lookAheadAtLocations",
	)
	defer span.End()

	if locations == nil {
		return
	}

	for _, location := range locations {
		// Get the next look ahead epoch
		epoch := phase0.Epoch(location.GetEthV2BeaconBlockAttesterSlashing().GetEpoch())

		sp, err := a.beacon.Node().Spec()
		if err != nil {
			a.log.WithError(err).WithField("epoch", epoch).Warn("Failed to look ahead at epoch")

			return
		}

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

	for _, slashing := range a.getAttesterSlashings(ctx, block) {
		event, err := a.createEvent(ctx, slashing, blockIdentifier)
		if err != nil {
			a.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for attester slashing %s", slashing.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (a *AttesterSlashingDeriver) getAttesterSlashings(ctx context.Context, block *spec.VersionedSignedBeaconBlock) []*xatuethv1.AttesterSlashingV2 {
	slashings := []*xatuethv1.AttesterSlashingV2{}

	attesterSlashings, err := block.AttesterSlashings()
	if err != nil {
		a.log.WithError(err).Error("Failed to obtain attester slashings")
	}

	for _, slashing := range attesterSlashings {
		slashings = append(slashings, &xatuethv1.AttesterSlashingV2{
			Attestation_1: convertIndexedAttestation(slashing.Attestation1),
			Attestation_2: convertIndexedAttestation(slashing.Attestation2),
		})
	}

	return slashings
}

func convertIndexedAttestation(attestation *phase0.IndexedAttestation) *xatuethv1.IndexedAttestationV2 {
	indicies := []*wrapperspb.UInt64Value{}

	for _, index := range attestation.AttestingIndices {
		indicies = append(indicies, &wrapperspb.UInt64Value{Value: index})
	}

	return &xatuethv1.IndexedAttestationV2{
		AttestingIndices: indicies,
		Data: &xatuethv1.AttestationDataV2{
			Slot:            &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Slot)},
			Index:           &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Index)},
			BeaconBlockRoot: attestation.Data.BeaconBlockRoot.String(),
			Source: &xatuethv1.CheckpointV2{
				Epoch: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Source.Epoch)},
				Root:  attestation.Data.Source.Root.String(),
			},
			Target: &xatuethv1.CheckpointV2{
				Epoch: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Target.Epoch)},
				Root:  attestation.Data.Target.Root.String(),
			},
		},
		Signature: attestation.Signature.String(),
	}
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
