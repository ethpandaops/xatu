package v2

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v5"
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
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
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
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v2/elaborated_attestation",
			"type":   ElaboratedAttestationDeriverName.String(),
		}),
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

func (b *ElaboratedAttestationDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
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

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
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

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

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
					b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}

			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
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
		attestationData, err := attestation.Data()
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain attestation data")
		}

		signature, err := attestation.Signature()
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain attestation signature")
		}

		// Handle different attestation versions
		switch attestation.Version {
		case spec.DataVersionPhase0, spec.DataVersionAltair, spec.DataVersionBellatrix, spec.DataVersionCapella, spec.DataVersionDeneb:
			// For pre-Electra attestations, each attestation can only have one committee
			indexes, indexErr := b.getAttestatingValidatorIndexesPhase0(ctx, attestation)
			if indexErr != nil {
				return nil, errors.Wrap(indexErr, "failed to get attestating validator indexes")
			}

			// Create a single elaborated attestation
			elaboratedAttestation := &xatuethv1.ElaboratedAttestation{
				Signature: signature.String(),
				Data: &xatuethv1.AttestationDataV2{
					Slot:            &wrapperspb.UInt64Value{Value: uint64(attestationData.Slot)},
					Index:           &wrapperspb.UInt64Value{Value: uint64(attestationData.Index)},
					BeaconBlockRoot: xatuethv1.RootAsString(attestationData.BeaconBlockRoot),
					Source: &xatuethv1.CheckpointV2{
						Epoch: &wrapperspb.UInt64Value{Value: uint64(attestationData.Source.Epoch)},
						Root:  xatuethv1.RootAsString(attestationData.Source.Root),
					},
					Target: &xatuethv1.CheckpointV2{
						Epoch: &wrapperspb.UInt64Value{Value: uint64(attestationData.Target.Epoch)},
						Root:  xatuethv1.RootAsString(attestationData.Target.Root),
					},
				},
				ValidatorIndexes: indexes,
			}

			//nolint:gosec // If we have that many attestations in a block we're cooked
			event, err := b.createEventFromElaboratedAttestation(ctx, elaboratedAttestation, uint64(positionInBlock), blockIdentifier)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create event for attestation %s", attestation.String())
			}

			events = append(events, event)

		default:
			// For Electra attestations, create multiple events (one per committee)
			// Get the committee bits (this indicates which committees are included in this attestation)
			committeeBits, err := attestation.CommitteeBits()
			if err != nil {
				return nil, errors.Wrap(err, "failed to obtain attestation committee bits")
			}

			// Get aggregation bits
			aggregationBits, err := attestation.AggregationBits()
			if err != nil {
				return nil, errors.Wrap(err, "failed to obtain attestation aggregation bits")
			}

			// Process each committee from the committee_bits
			committeeIndices := committeeBits.BitIndices()
			committeeOffset := 0

			for _, committeeIdx := range committeeIndices {
				// Get the committee information
				epoch := b.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(attestationData.Slot))

				epochCommittees, err := b.beacon.Duties().FetchBeaconCommittee(ctx, phase0.Epoch(epoch.Number()))
				if err != nil {
					return nil, errors.Wrap(err, "failed to get committees for epoch")
				}

				// Find the committee matching our current slot and index
				var committee *v1.BeaconCommittee

				for _, c := range epochCommittees {
					//nolint:gosec // This is capped at 64 committees in the spec
					if c.Slot == attestationData.Slot && c.Index == phase0.CommitteeIndex(committeeIdx) {
						committee = c

						break
					}
				}

				if committee == nil {
					return nil, errors.New(fmt.Sprintf("committee %d in slot %d not found", committeeIdx, attestationData.Slot))
				}

				committeeSize := len(committee.Validators)

				// Create committee-specific validator indexes array
				committeeValidatorIndexes := []*wrapperspb.UInt64Value{}

				// For each validator position in this committee
				for i := range committeeSize {
					// Calculate the bit position in the aggregation_bits
					aggregationBitPosition := committeeOffset + i

					// Check if this position is valid and set
					//nolint:gosec // This is capped at 64 committees in the spec
					if uint64(aggregationBitPosition) < aggregationBits.Len() && aggregationBits.BitAt(uint64(aggregationBitPosition)) {
						validatorIndex := committee.Validators[i]
						committeeValidatorIndexes = append(committeeValidatorIndexes, wrapperspb.UInt64(uint64(validatorIndex)))
					}
				}

				// Create an elaborated attestation for this committee
				elaboratedAttestation := &xatuethv1.ElaboratedAttestation{
					Signature: signature.String(),
					Data: &xatuethv1.AttestationDataV2{
						Slot: &wrapperspb.UInt64Value{Value: uint64(attestationData.Slot)},
						//nolint:gosec // This is capped at 64 committees in the spec
						Index:           &wrapperspb.UInt64Value{Value: uint64(committeeIdx)}, // Use the committee index from committee_bits
						BeaconBlockRoot: xatuethv1.RootAsString(attestationData.BeaconBlockRoot),
						Source: &xatuethv1.CheckpointV2{
							Epoch: &wrapperspb.UInt64Value{Value: uint64(attestationData.Source.Epoch)},
							Root:  xatuethv1.RootAsString(attestationData.Source.Root),
						},
						Target: &xatuethv1.CheckpointV2{
							Epoch: &wrapperspb.UInt64Value{Value: uint64(attestationData.Target.Epoch)},
							Root:  xatuethv1.RootAsString(attestationData.Target.Root),
						},
					},
					ValidatorIndexes: committeeValidatorIndexes,
				}

				//nolint:gosec // If we have that many attestations in a block we're cooked
				event, err := b.createEventFromElaboratedAttestation(ctx, elaboratedAttestation, uint64(positionInBlock), blockIdentifier)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to create event for attestation %s committee %d", attestation.String(), committeeIdx)
				}

				events = append(events, event)

				// Update offset for the next committee
				committeeOffset += committeeSize
			}
		}
	}

	return events, nil
}

func (b *ElaboratedAttestationDeriver) getAttestatingValidatorIndexesPhase0(ctx context.Context, attestation *spec.VersionedAttestation) ([]*wrapperspb.UInt64Value, error) {
	indexes := []*wrapperspb.UInt64Value{}

	attestationData, err := attestation.Data()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation data")
	}

	epoch := b.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(attestationData.Slot))

	bitIndices, err := attestation.AggregationBits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation aggregation bits")
	}

	for _, position := range bitIndices.BitIndices() {
		validatorIndex, err := b.beacon.Duties().GetValidatorIndex(
			ctx,
			phase0.Epoch(epoch.Number()),
			attestationData.Slot,
			attestationData.Index,
			//nolint:gosec // This is capped at 64 committees in the spec
			uint64(position),
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get validator index for position %d", position)
		}

		indexes = append(indexes, wrapperspb.UInt64(uint64(validatorIndex)))
	}

	return indexes, nil
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
