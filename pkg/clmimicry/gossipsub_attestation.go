package clmimicry

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/probe-lab/hermes/host"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethtypes "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (m *Mimicry) handleGossipAttestation(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent, payload map[string]any) error {
	// Extract attestation data
	eAttestation := &ethtypes.Attestation{
		Data: &ethtypes.AttestationData{},
	}

	if slot, ok := payload["Slot"].(primitives.Slot); ok {
		eAttestation.Data.Slot = slot
	} else {
		return fmt.Errorf("invalid slot")
	}

	if committeeIndex, ok := payload["CommIdx"].(primitives.CommitteeIndex); ok {
		eAttestation.Data.CommitteeIndex = committeeIndex
	} else {
		return fmt.Errorf("invalid committee index")
	}

	if beaconBlockRoot, ok := payload["BeaconBlockRoot"].([]byte); ok {
		eAttestation.Data.BeaconBlockRoot = beaconBlockRoot
	} else {
		return fmt.Errorf("invalid beacon block root")
	}

	if source, ok := payload["Source"].(*ethtypes.Checkpoint); ok {
		eAttestation.Data.Source = source
	} else {
		return fmt.Errorf("invalid source")
	}

	if target, ok := payload["Target"].(*ethtypes.Checkpoint); ok {
		eAttestation.Data.Target = target
	} else {
		return fmt.Errorf("invalid target")
	}

	attestation := &v1.Attestation{
		AggregationBits: string(""),
		Data: &v1.AttestationData{
			Slot:            uint64(eAttestation.Data.Slot),
			BeaconBlockRoot: fmt.Sprintf("0x%x", eAttestation.Data.BeaconBlockRoot),
			Source: &v1.Checkpoint{
				Epoch: uint64(eAttestation.Data.Source.Epoch),
				Root:  fmt.Sprintf("0x%x", eAttestation.Data.Source.Root),
			},
			Target: &v1.Checkpoint{
				Epoch: uint64(eAttestation.Data.Target.Epoch),
				Root:  fmt.Sprintf("0x%x", eAttestation.Data.Target.Root),
			},
			Index: uint64(eAttestation.Data.CommitteeIndex),
		},
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := m.createAdditionalGossipSubAttestationData(ctx, payload, eAttestation, event)
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

func (m *Mimicry) createAdditionalGossipSubAttestationData(ctx context.Context,
	payload map[string]any,
	attestation *ethtypes.Attestation,
	event *host.TraceEvent,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconAttestationData, error) {
	wallclockSlot, wallclockEpoch, err := m.ethereum.Metadata().Wallclock().Now()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	// Add Clock Drift
	timestampAdjusted := event.Timestamp.Add(m.clockDrift)

	attestionSlot := m.ethereum.Metadata().Wallclock().Slots().FromNumber(uint64(attestation.Data.Slot))
	epoch := m.ethereum.Metadata().Wallclock().Epochs().FromSlot(uint64(attestation.Data.Slot))

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

	targetEpoch := m.ethereum.Metadata().Wallclock().Epochs().FromNumber(uint64(attestation.Data.Target.Epoch))
	extra.Target = &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconAttestationTargetData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: targetEpoch.Number()},
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	sourceEpoch := m.ethereum.Metadata().Wallclock().Epochs().FromNumber(uint64(attestation.Data.Source.Epoch))
	extra.Source = &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconAttestationSourceData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: sourceEpoch.Number()},
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	peerID, ok := payload["PeerID"].(string)
	if ok {
		extra.Metadata = &libp2p.TraceEventMetadata{
			PeerId: wrapperspb.String(peerID),
		}
	}

	topic, ok := payload["Topic"].(string)
	if ok {
		extra.Topic = wrapperspb.String(topic)
	}

	msgID, ok := payload["MsgID"].(string)
	if ok {
		extra.MessageId = wrapperspb.String(msgID)
	}

	msgSize, ok := payload["MsgSize"].(int)
	if ok {
		extra.MessageSize = wrapperspb.UInt32(uint32(msgSize))
	}

	// If the attestation is unaggreated, we can append the validator position within the committee
	position, ok := payload["AggregatePos"].(int)
	if ok {
		validatorIndex, err := m.ethereum.Duties().GetValidatorIndex(
			phase0.Epoch(epoch.Number()),
			phase0.Slot(attestation.Data.Slot),
			phase0.CommitteeIndex(attestation.Data.CommitteeIndex),
			uint64(position),
		)
		if err == nil {
			extra.AttestingValidator = &xatu.AttestingValidatorV2{
				CommitteeIndex: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.CommitteeIndex)},
				Index:          &wrapperspb.UInt64Value{Value: uint64(validatorIndex)},
			}
		}
	}

	return extra, nil
}
