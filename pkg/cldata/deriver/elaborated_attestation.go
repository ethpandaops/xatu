package deriver

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/iterator"
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

// ElaboratedAttestationDeriverConfig is the configuration for the ElaboratedAttestationDeriver.
type ElaboratedAttestationDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// ElaboratedAttestationDeriver extracts elaborated attestations from beacon blocks.
type ElaboratedAttestationDeriver struct {
	log               logrus.FieldLogger
	cfg               *ElaboratedAttestationDeriverConfig
	iterator          iterator.Iterator
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
}

// NewElaboratedAttestationDeriver creates a new ElaboratedAttestationDeriver.
func NewElaboratedAttestationDeriver(
	log logrus.FieldLogger,
	config *ElaboratedAttestationDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctx cldata.ContextProvider,
) *ElaboratedAttestationDeriver {
	return &ElaboratedAttestationDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/elaborated_attestation",
			"type":   ElaboratedAttestationDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctx,
	}
}

// CannonType returns the cannon type of the deriver.
func (d *ElaboratedAttestationDeriver) CannonType() xatu.CannonType {
	return ElaboratedAttestationDeriverName
}

// Name returns the name of the deriver.
func (d *ElaboratedAttestationDeriver) Name() string {
	return ElaboratedAttestationDeriverName.String()
}

// ActivationFork returns the fork at which the deriver is activated.
func (d *ElaboratedAttestationDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// OnEventsDerived registers a callback for when events are derived.
func (d *ElaboratedAttestationDeriver) OnEventsDerived(
	_ context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	d.onEventsCallbacks = append(d.onEventsCallbacks, fn)
}

// Start starts the deriver.
func (d *ElaboratedAttestationDeriver) Start(ctx context.Context) error {
	if !d.cfg.Enabled {
		d.log.Info("Elaborated attestation deriver disabled")

		return nil
	}

	d.log.Info("Elaborated attestation deriver enabled")

	if err := d.iterator.Start(ctx, d.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	d.run(ctx)

	return nil
}

// Stop stops the deriver.
func (d *ElaboratedAttestationDeriver) Stop(_ context.Context) error {
	return nil
}

func (d *ElaboratedAttestationDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", d.Name()),
					trace.WithAttributes(
						attribute.String("network", d.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := d.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position.
				position, err := d.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Process the epoch
				events, err := d.processEpoch(ctx, position.Epoch)
				if err != nil {
					d.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				d.lookAhead(ctx, position.LookAheadEpochs)

				// Send the events
				for _, fn := range d.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := d.iterator.UpdateLocation(ctx, position); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					d.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				d.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

func (d *ElaboratedAttestationDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ElaboratedAttestationDeriver.processEpoch",
		//nolint:gosec // epoch values won't overflow int64
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	allEvents := make([]*xatu.DecoratedEvent, 0)

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		d.log.WithError(err).WithField("epoch", epoch).Warn("Failed to look ahead at epoch")

		return nil, err
	}

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := d.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (d *ElaboratedAttestationDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ElaboratedAttestationDeriver.processSlot",
		//nolint:gosec // slot values won't overflow int64
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := d.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	events, err := d.getElaboratedAttestations(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get elaborated attestations for slot %d", slot)
	}

	return events, nil
}

// lookAhead attempts to pre-load any blocks that might be required for the epochs that are coming up.
func (d *ElaboratedAttestationDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ElaboratedAttestationDeriver.lookAhead",
	)
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		d.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			d.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (d *ElaboratedAttestationDeriver) getElaboratedAttestations(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
) ([]*xatu.DecoratedEvent, error) {
	blockAttestations, err := block.Attestations()
	if err != nil {
		return nil, err
	}

	blockIdentifier, err := GetBlockIdentifier(block, d.ctx.Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for block")
	}

	events := make([]*xatu.DecoratedEvent, 0, len(blockAttestations))

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
		case spec.DataVersionPhase0, spec.DataVersionAltair, spec.DataVersionBellatrix,
			spec.DataVersionCapella, spec.DataVersionDeneb:
			// For pre-Electra attestations, each attestation can only have one committee
			indexes, indexErr := d.getAttestingValidatorIndexesPhase0(ctx, attestation)
			if indexErr != nil {
				return nil, errors.Wrap(indexErr, "failed to get attesting validator indexes")
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
			event, eventErr := d.createEventFromElaboratedAttestation(
				ctx,
				elaboratedAttestation,
				uint64(positionInBlock),
				blockIdentifier,
			)
			if eventErr != nil {
				return nil, errors.Wrapf(eventErr, "failed to create event for attestation %s", attestation.String())
			}

			events = append(events, event)

		default:
			// For Electra attestations, create multiple events (one per committee)
			electraEvents, electraErr := d.processElectraAttestation(
				ctx,
				attestation,
				attestationData,
				&signature,
				positionInBlock,
				blockIdentifier,
			)
			if electraErr != nil {
				return nil, electraErr
			}

			events = append(events, electraEvents...)
		}
	}

	return events, nil
}

func (d *ElaboratedAttestationDeriver) processElectraAttestation(
	ctx context.Context,
	attestation *spec.VersionedAttestation,
	attestationData *phase0.AttestationData,
	signature *phase0.BLSSignature,
	positionInBlock int,
	blockIdentifier *xatu.BlockIdentifier,
) ([]*xatu.DecoratedEvent, error) {
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
	events := make([]*xatu.DecoratedEvent, 0, len(committeeIndices))

	for _, committeeIdx := range committeeIndices {
		// Get the committee information
		epoch := d.ctx.Wallclock().Epochs().FromSlot(uint64(attestationData.Slot))

		epochCommittees, err := d.beacon.FetchBeaconCommittee(ctx, phase0.Epoch(epoch.Number()))
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
			return nil, fmt.Errorf("committee %d in slot %d not found", committeeIdx, attestationData.Slot)
		}

		committeeSize := len(committee.Validators)

		// Create committee-specific validator indexes array
		committeeValidatorIndexes := make([]*wrapperspb.UInt64Value, 0, committeeSize)

		// For each validator position in this committee
		for i := 0; i < committeeSize; i++ {
			// Calculate the bit position in the aggregation_bits
			aggregationBitPosition := committeeOffset + i

			// Check if this position is valid and set
			//nolint:gosec // This is capped at 64 committees in the spec
			if uint64(aggregationBitPosition) < aggregationBits.Len() &&
				aggregationBits.BitAt(uint64(aggregationBitPosition)) {
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
				Index:           &wrapperspb.UInt64Value{Value: uint64(committeeIdx)},
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
		event, err := d.createEventFromElaboratedAttestation(
			ctx,
			elaboratedAttestation,
			uint64(positionInBlock),
			blockIdentifier,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to create event for attestation %s committee %d",
				attestation.String(),
				committeeIdx,
			)
		}

		events = append(events, event)

		// Update offset for the next committee
		committeeOffset += committeeSize
	}

	return events, nil
}

func (d *ElaboratedAttestationDeriver) getAttestingValidatorIndexesPhase0(
	ctx context.Context,
	attestation *spec.VersionedAttestation,
) ([]*wrapperspb.UInt64Value, error) {
	attestationData, err := attestation.Data()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation data")
	}

	epoch := d.ctx.Wallclock().Epochs().FromSlot(uint64(attestationData.Slot))

	bitIndices, err := attestation.AggregationBits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation aggregation bits")
	}

	positions := bitIndices.BitIndices()
	indexes := make([]*wrapperspb.UInt64Value, 0, len(positions))

	for _, position := range positions {
		validatorIndex, err := d.beacon.GetValidatorIndex(
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

func (d *ElaboratedAttestationDeriver) createEventFromElaboratedAttestation(
	_ context.Context,
	attestation *xatuethv1.ElaboratedAttestation,
	positionInBlock uint64,
	blockIdentifier *xatu.BlockIdentifier,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := d.ctx.CreateClientMeta(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client metadata")
	}

	// Make a clone of the metadata
	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
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

	attestationSlot := d.ctx.Wallclock().Slots().FromNumber(attestation.Data.Slot.Value)
	epoch := d.ctx.Wallclock().Epochs().FromSlot(attestationSlot.Number())

	// Build out the target section
	targetEpoch := d.ctx.Wallclock().Epochs().FromNumber(attestation.Data.Target.Epoch.GetValue())
	target := &xatu.ClientMeta_AdditionalEthV1AttestationTargetV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: targetEpoch.Number()},
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	// Build out the source section
	sourceEpoch := d.ctx.Wallclock().Epochs().FromNumber(attestation.Data.Source.Epoch.GetValue())
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
