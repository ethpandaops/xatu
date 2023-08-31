package v2

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type AttesterSlashingDeriver struct {
	log logrus.FieldLogger
}

func NewAttesterSlashingDeriver(log logrus.FieldLogger) *AttesterSlashingDeriver {
	return &AttesterSlashingDeriver{
		log: log.WithField("module", "cannon/event/beacon/eth/v2/attester_slashing"),
	}
}

func (a *AttesterSlashingDeriver) Process(ctx context.Context, metadata *BeaconBlockMetadata, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error) {
	events := []*xatu.DecoratedEvent{}

	for _, slashing := range a.getAttesterSlashings(ctx, block) {
		event, err := a.createEvent(ctx, metadata, slashing)
		if err != nil {
			a.log.WithError(err).Error("Failed to create event")

			continue
		}

		events = append(events, event)
	}

	return events, nil
}

func (a *AttesterSlashingDeriver) Name() string {
	return "AttesterSlashing"
}

func (a *AttesterSlashingDeriver) getAttesterSlashings(ctx context.Context, block *spec.VersionedSignedBeaconBlock) []*xatuethv1.AttesterSlashingV2 {
	slashings := []*xatuethv1.AttesterSlashingV2{}

	switch block.Version {
	case spec.DataVersionPhase0:
		for _, slashing := range block.Phase0.Message.Body.AttesterSlashings {
			slashings = append(slashings, &xatuethv1.AttesterSlashingV2{
				Attestation_1: convertIndexedAttestation(slashing.Attestation1),
				Attestation_2: convertIndexedAttestation(slashing.Attestation2),
			})
		}
	case spec.DataVersionAltair:
		for _, slashing := range block.Altair.Message.Body.AttesterSlashings {
			slashings = append(slashings, &xatuethv1.AttesterSlashingV2{
				Attestation_1: convertIndexedAttestation(slashing.Attestation1),
				Attestation_2: convertIndexedAttestation(slashing.Attestation2),
			})
		}
	case spec.DataVersionBellatrix:
		for _, slashing := range block.Bellatrix.Message.Body.AttesterSlashings {
			slashings = append(slashings, &xatuethv1.AttesterSlashingV2{
				Attestation_1: convertIndexedAttestation(slashing.Attestation1),
				Attestation_2: convertIndexedAttestation(slashing.Attestation2),
			})
		}
	case spec.DataVersionCapella:
		for _, slashing := range block.Capella.Message.Body.AttesterSlashings {
			slashings = append(slashings, &xatuethv1.AttesterSlashingV2{
				Attestation_1: convertIndexedAttestation(slashing.Attestation1),
				Attestation_2: convertIndexedAttestation(slashing.Attestation2),
			})
		}
	}

	return slashings
}

func convertIndexedAttestation(attestation *phase0.IndexedAttestation) *xatuethv1.IndexedAttestationV2 {
	indicies := []*wrapperspb.UInt64Value{}

	for _, index := range attestation.AttestingIndices {
		indicies = append(indicies, &wrapperspb.UInt64Value{Value: uint64(index)})
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

func (a *AttesterSlashingDeriver) createEvent(ctx context.Context, metadata *BeaconBlockMetadata, slashing *xatuethv1.AttesterSlashingV2) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
			DateTime: timestamppb.New(metadata.Now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata.ClientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: slashing,
		},
	}

	blockIdentifier, err := metadata.BlockIdentifier()
	if err != nil {
		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockAttesterSlashing{
		EthV2BeaconBlockAttesterSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAttesterSlashingData{
			Block: blockIdentifier,
		},
	}

	return decoratedEvent, nil
}
