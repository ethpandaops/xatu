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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ElaboratedAttestationDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION
)

type ElaboratedAttestationDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

type ElaboratedAttestationDeriver struct {
	log               logrus.FieldLogger
	cfg               *ElaboratedAttestationDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewElaboratedAttestationDeriver(log logrus.FieldLogger, config *ElaboratedAttestationDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ElaboratedAttestationDeriver {
	return &ElaboratedAttestationDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v2/elaborated_attestation"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *ElaboratedAttestationDeriver) CannonType() xatu.CannonType {
	return ElaboratedAttestationDeriverName
}

func (b *ElaboratedAttestationDeriver) Name() string {
	return ElaboratedAttestationDeriverName.String()
}

func (b *ElaboratedAttestationDeriver) ActivationFork() string {
	return ethereum.ForkNamePhase0
}

func (b *ElaboratedAttestationDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *ElaboratedAttestationDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Elaborated attestation deriver disabled")

		return nil
	}

	b.log.Info("Elaborated attestation deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *ElaboratedAttestationDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *ElaboratedAttestationDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() error {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Get the next position.
				position, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					span.SetStatus(codes.Error, err.Error())

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

func (b *ElaboratedAttestationDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ElaboratedAttestationDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	allEvents := []*xatu.DecoratedEvent{}

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).WithField("epoch", epoch).Warn("Failed to look ahead at epoch")

		return nil, err
	}

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

func (b *ElaboratedAttestationDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ElaboratedAttestationDeriver.processSlot",
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	events, err := b.getElaboratedAttestations(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get elaborated attestations for slot %d", slot)
	}

	return events, nil
}

// lookAhead attempts to pre-load any blocks that might be required for the epochs that are coming up.
func (b *ElaboratedAttestationDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ElaboratedAttestationDeriver.lookAhead",
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).Warn("Failed to look ahead at epoch")

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

func (b *ElaboratedAttestationDeriver) getElaboratedAttestations(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error) {
	blockAttestations, err := block.Attestations()
	if err != nil {
		return nil, err
	}

	blockIdentifier, err := GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for block")
	}

	events := []*xatu.DecoratedEvent{}

	for positionInBlock, attestation := range blockAttestations {
		epoch := b.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(attestation.Data.Slot))

		// Get all the validator indexes of those who signed the attestation
		indexes := []*wrapperspb.UInt64Value{}

		for _, position := range attestation.AggregationBits.BitIndices() {
			validatorIndex, err := b.beacon.Duties().GetValidatorIndex(
				phase0.Epoch(epoch.Number()),
				attestation.Data.Slot,
				attestation.Data.Index,
				uint64(position),
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get validator index for position %d", position)
			}

			indexes = append(indexes, wrapperspb.UInt64(uint64(validatorIndex)))
		}

		elaboratedAttestation := &xatuethv1.ElaboratedAttestation{
			Signature: attestation.Signature.String(),
			Data: &xatuethv1.AttestationDataV2{
				Slot:            &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Slot)},
				Index:           &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Index)},
				BeaconBlockRoot: xatuethv1.RootAsString(attestation.Data.BeaconBlockRoot),
				Source: &xatuethv1.CheckpointV2{
					Epoch: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Source.Epoch)},
					Root:  xatuethv1.RootAsString(attestation.Data.Source.Root),
				},
				Target: &xatuethv1.CheckpointV2{
					Epoch: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Target.Epoch)},
					Root:  xatuethv1.RootAsString(attestation.Data.Target.Root),
				},
			},
			ValidatorIndexes: indexes,
		}

		event, err := b.createEventFromElaboratedAttestation(ctx, elaboratedAttestation, uint64(positionInBlock), blockIdentifier)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create event for attestation %s", attestation.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *ElaboratedAttestationDeriver) createEventFromElaboratedAttestation(ctx context.Context, attestation *xatuethv1.ElaboratedAttestation, positionInBlock uint64, blockIdentifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockElaboratedAttestation{
			EthV2BeaconBlockElaboratedAttestation: attestation,
		},
	}

	attestationSlot := b.beacon.Metadata().Wallclock().Slots().FromNumber(attestation.Data.Slot.Value)
	epoch := b.beacon.Metadata().Wallclock().Epochs().FromSlot(attestationSlot.Number())

	// Build out the target section
	targetEpoch := b.beacon.Metadata().Wallclock().Epochs().FromNumber(attestation.Data.Target.Epoch.GetValue())
	target := &xatu.ClientMeta_AdditionalEthV1AttestationTargetV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: targetEpoch.Number()},
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	// Build out the source section
	sourceEpoch := b.beacon.Metadata().Wallclock().Epochs().FromNumber(attestation.Data.Source.Epoch.GetValue())
	source := &xatu.ClientMeta_AdditionalEthV1AttestationSourceV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: sourceEpoch.Number()},
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockElaboratedAttestation{
		EthV2BeaconBlockElaboratedAttestation: &xatu.ClientMeta_AdditionalEthV2BeaconBlockElaboratedAttestationData{
			Block:           blockIdentifier,
			PositionInBlock: wrapperspb.UInt64(positionInBlock),
			Slot: &xatu.SlotV2{
				Number:        &wrapperspb.UInt64Value{Value: attestationSlot.Number()},
				StartDateTime: timestamppb.New(attestationSlot.TimeWindow().Start()),
			},
			Epoch: &xatu.EpochV2{
				Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
				StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
			},
			Source: source,
			Target: target,
		},
	}

	return decoratedEvent, nil
}
