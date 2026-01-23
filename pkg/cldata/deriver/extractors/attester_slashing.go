package extractors

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "attester_slashing",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractAttesterSlashings,
	})
}

// ExtractAttesterSlashings extracts attester slashing events from a beacon block.
func ExtractAttesterSlashings(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	_ cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	slashings, err := block.AttesterSlashings()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attester slashings")
	}

	if len(slashings) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(slashings))

	for _, slashing := range slashings {
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

		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING)
		if err != nil {
			return nil, err
		}

		event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: &xatuethv1.AttesterSlashingV2{
				Attestation_1: indexedAttestation1,
				Attestation_2: indexedAttestation2,
			},
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAttesterSlashingData{
				Block: blockID,
			},
		}

		events = append(events, event)
	}

	return events, nil
}

// convertIndexedAttestation converts a VersionedIndexedAttestation to an IndexedAttestationV2.
func convertIndexedAttestation(attestation *spec.VersionedIndexedAttestation) (*xatuethv1.IndexedAttestationV2, error) {
	indices := make([]*wrapperspb.UInt64Value, 0)

	atIndices, err := attestation.AttestingIndices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attesting indices")
	}

	for _, index := range atIndices {
		indices = append(indices, &wrapperspb.UInt64Value{Value: index})
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
		AttestingIndices: indices,
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
