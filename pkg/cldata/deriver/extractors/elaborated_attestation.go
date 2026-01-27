package extractors

import (
	"context"
	"fmt"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "elaborated_attestation",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractElaboratedAttestations,
	})
}

// ExtractElaboratedAttestations extracts elaborated attestation events from a beacon block.
func ExtractElaboratedAttestations(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	blockAttestations, err := block.Attestations()
	if err != nil {
		return nil, err
	}

	if len(blockAttestations) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	builder := deriver.NewEventBuilder(ctxProvider)
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
			indexes, indexErr := getAttestingValidatorIndexesPhase0(ctx, attestation, beacon, ctxProvider)
			if indexErr != nil {
				return nil, errors.Wrap(indexErr, "failed to get attesting validator indexes")
			}

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

			//nolint:gosec // positionInBlock bounded by attestations per block
			event, eventErr := createElaboratedAttestationEvent(
				ctx,
				builder,
				elaboratedAttestation,
				uint64(positionInBlock),
				blockID,
				ctxProvider,
			)
			if eventErr != nil {
				return nil, errors.Wrapf(eventErr, "failed to create event for attestation %s", attestation.String())
			}

			events = append(events, event)

		default:
			// For Electra attestations, create multiple events (one per committee)
			electraEvents, electraErr := processElectraAttestation(
				ctx,
				builder,
				attestation,
				attestationData,
				&signature,
				positionInBlock,
				blockID,
				beacon,
				ctxProvider,
			)
			if electraErr != nil {
				return nil, electraErr
			}

			events = append(events, electraEvents...)
		}
	}

	return events, nil
}

func processElectraAttestation(
	ctx context.Context,
	builder *deriver.EventBuilder,
	attestation *spec.VersionedAttestation,
	attestationData *phase0.AttestationData,
	signature *phase0.BLSSignature,
	positionInBlock int,
	blockID *xatu.BlockIdentifier,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	committeeBits, err := attestation.CommitteeBits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation committee bits")
	}

	aggregationBits, err := attestation.AggregationBits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation aggregation bits")
	}

	committeeIndices := committeeBits.BitIndices()
	committeeOffset := 0
	events := make([]*xatu.DecoratedEvent, 0, len(committeeIndices))

	for _, committeeIdx := range committeeIndices {
		epoch := ctxProvider.Wallclock().Epochs().FromSlot(uint64(attestationData.Slot))

		epochCommittees, err := beacon.FetchBeaconCommittee(ctx, phase0.Epoch(epoch.Number()))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get committees for epoch")
		}

		var committee *v1.BeaconCommittee

		for _, c := range epochCommittees {
			//nolint:gosec // committeeIdx capped at 64 committees in spec
			if c.Slot == attestationData.Slot && c.Index == phase0.CommitteeIndex(committeeIdx) {
				committee = c

				break
			}
		}

		if committee == nil {
			return nil, fmt.Errorf("committee %d in slot %d not found", committeeIdx, attestationData.Slot)
		}

		committeeSize := len(committee.Validators)
		committeeValidatorIndexes := make([]*wrapperspb.UInt64Value, 0, committeeSize)

		for i := 0; i < committeeSize; i++ {
			aggregationBitPosition := committeeOffset + i

			//nolint:gosec // aggregationBitPosition bounded by committee size
			if uint64(aggregationBitPosition) < aggregationBits.Len() &&
				aggregationBits.BitAt(uint64(aggregationBitPosition)) {
				validatorIndex := committee.Validators[i]
				committeeValidatorIndexes = append(committeeValidatorIndexes, wrapperspb.UInt64(uint64(validatorIndex)))
			}
		}

		elaboratedAttestation := &xatuethv1.ElaboratedAttestation{
			Signature: signature.String(),
			Data: &xatuethv1.AttestationDataV2{
				Slot: &wrapperspb.UInt64Value{Value: uint64(attestationData.Slot)},
				//nolint:gosec // committeeIdx capped at 64 committees in spec
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

		//nolint:gosec // positionInBlock bounded by attestations per block
		event, err := createElaboratedAttestationEvent(
			ctx,
			builder,
			elaboratedAttestation,
			uint64(positionInBlock),
			blockID,
			ctxProvider,
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
		committeeOffset += committeeSize
	}

	return events, nil
}

func getAttestingValidatorIndexesPhase0(
	ctx context.Context,
	attestation *spec.VersionedAttestation,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*wrapperspb.UInt64Value, error) {
	attestationData, err := attestation.Data()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation data")
	}

	epoch := ctxProvider.Wallclock().Epochs().FromSlot(uint64(attestationData.Slot))

	bitIndices, err := attestation.AggregationBits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation aggregation bits")
	}

	positions := bitIndices.BitIndices()
	indexes := make([]*wrapperspb.UInt64Value, 0, len(positions))

	for _, position := range positions {
		validatorIndex, err := beacon.GetValidatorIndex(
			ctx,
			phase0.Epoch(epoch.Number()),
			attestationData.Slot,
			attestationData.Index,
			//nolint:gosec // position bounded by committee size
			uint64(position),
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get validator index for position %d", position)
		}

		indexes = append(indexes, wrapperspb.UInt64(uint64(validatorIndex)))
	}

	return indexes, nil
}

func createElaboratedAttestationEvent(
	ctx context.Context,
	builder *deriver.EventBuilder,
	attestation *xatuethv1.ElaboratedAttestation,
	positionInBlock uint64,
	blockID *xatu.BlockIdentifier,
	ctxProvider cldata.ContextProvider,
) (*xatu.DecoratedEvent, error) {
	event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION)
	if err != nil {
		return nil, err
	}

	event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockElaboratedAttestation{
		EthV2BeaconBlockElaboratedAttestation: attestation,
	}

	attestationSlot := ctxProvider.Wallclock().Slots().FromNumber(attestation.Data.Slot.Value)
	epoch := ctxProvider.Wallclock().Epochs().FromSlot(attestationSlot.Number())

	targetEpoch := ctxProvider.Wallclock().Epochs().FromNumber(attestation.Data.Target.Epoch.GetValue())
	target := &xatu.ClientMeta_AdditionalEthV1AttestationTargetV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: targetEpoch.Number()},
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	sourceEpoch := ctxProvider.Wallclock().Epochs().FromNumber(attestation.Data.Source.Epoch.GetValue())
	source := &xatu.ClientMeta_AdditionalEthV1AttestationSourceV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: sourceEpoch.Number()},
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockElaboratedAttestation{
		EthV2BeaconBlockElaboratedAttestation: &xatu.ClientMeta_AdditionalEthV2BeaconBlockElaboratedAttestationData{
			Block:           blockID,
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

	return event, nil
}
