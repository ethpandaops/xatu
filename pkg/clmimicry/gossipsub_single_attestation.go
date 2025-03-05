package clmimicry

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"
	ethtypes "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (m *Mimicry) handleGossipSingleAttestation(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent,
	payload *eth.TraceEventSingleAttestation,
) error {
	if payload.SingleAttestation == nil || payload.SingleAttestation.GetData() == nil {
		return fmt.Errorf("handleGossipSingleAttestation() called with nil attestation")
	}

	attestationData := payload.SingleAttestation.GetData()

	attestation := &v1.Attestation{
		AggregationBits: string(""),
		Data: &v1.AttestationData{
			Slot:            uint64(attestationData.GetSlot()),
			BeaconBlockRoot: fmt.Sprintf("0x%x", attestationData.GetBeaconBlockRoot()),
			Source: &v1.Checkpoint{
				Epoch: uint64(attestationData.GetSource().GetEpoch()),
				Root:  fmt.Sprintf("0x%x", attestationData.GetSource().GetRoot()),
			},
			Target: &v1.Checkpoint{
				Epoch: uint64(attestationData.GetTarget().GetEpoch()),
				Root:  fmt.Sprintf("0x%x", attestationData.GetTarget().GetRoot()),
			},
			Index: uint64(attestationData.GetCommitteeIndex()),
		},
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := m.createAdditionalGossipSubSingleAttestationData(payload, attestationData, event)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubBeaconAttestation{
		Libp2PTraceGossipsubBeaconAttestation: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconAttestation{
			Libp2PTraceGossipsubBeaconAttestation: attestation,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (m *Mimicry) createAdditionalGossipSubSingleAttestationData(
	payload *eth.TraceEventSingleAttestation,
	attestationData *ethtypes.AttestationData,
	event *host.TraceEvent,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconAttestationData, error) {
	wallclockSlot, wallclockEpoch, err := m.ethereum.Metadata().Wallclock().Now()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	// Add Clock Drift
	timestampAdjusted := event.Timestamp.Add(m.clockDrift)

	attestionSlot := m.ethereum.Metadata().Wallclock().Slots().FromNumber(uint64(attestationData.GetSlot()))
	epoch := m.ethereum.Metadata().Wallclock().Epochs().FromSlot(uint64(attestationData.GetSlot()))

	extra := &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconAttestationData{
		WallclockSlot: &xatu.SlotV2{
			Number:        &wrapperspb.UInt64Value{Value: wallclockSlot.Number()},
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: wallclockEpoch.Number()},
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
		Slot: &xatu.SlotV2{
			Number:        &wrapperspb.UInt64Value{Value: attestionSlot.Number()},
			StartDateTime: timestamppb.New(attestionSlot.TimeWindow().Start()),
		},
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
		Propagation: &xatu.PropagationV2{
			SlotStartDiff: &wrapperspb.UInt64Value{
				Value: uint64(timestampAdjusted.Sub(attestionSlot.TimeWindow().Start()).Milliseconds()),
			},
		},
	}

	targetEpoch := m.ethereum.Metadata().Wallclock().Epochs().FromNumber(uint64(attestationData.GetTarget().GetEpoch()))
	extra.Target = &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconAttestationTargetData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: targetEpoch.Number()},
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	sourceEpoch := m.ethereum.Metadata().Wallclock().Epochs().FromNumber(uint64(attestationData.GetSource().GetEpoch()))
	extra.Source = &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconAttestationSourceData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: sourceEpoch.Number()},
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	extra.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(payload.PeerID)}
	extra.Topic = wrapperspb.String(payload.Topic)
	extra.MessageId = wrapperspb.String(payload.MsgID)
	extra.MessageSize = wrapperspb.UInt32(uint32(payload.MsgSize))

	// If the attestation is not aggregated, we can append the validator position within the committee.
	if payload.SingleAttestation.GetAggregationBits().Count() == 1 {
		validatorIndex, err := m.ethereum.Duties().GetValidatorIndex(
			phase0.Epoch(epoch.Number()),
			phase0.Slot(attestationData.GetSlot()),
			phase0.CommitteeIndex(attestationData.GetCommitteeIndex()),
			uint64(payload.SingleAttestation.GetAggregationBits().BitIndices()[0]),
		)
		if err == nil {
			extra.AttestingValidator = &xatu.AttestingValidatorV2{
				CommitteeIndex: &wrapperspb.UInt64Value{Value: uint64(attestationData.GetCommitteeIndex())},
				Index:          &wrapperspb.UInt64Value{Value: uint64(validatorIndex)},
			}
		}
	}

	return extra, nil
}
