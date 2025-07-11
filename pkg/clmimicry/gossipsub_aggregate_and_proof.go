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
	"google.golang.org/protobuf/proto"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func (m *Mimicry) handleGossipAggregateAndProof(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent,
	payload any,
) error {
	switch evt := payload.(type) {
	case *eth.TraceEventSignedAggregateAttestationAndProof:
		return m.handleAggregateAndProofFromAttestation(ctx, clientMeta, event, evt)
	case *eth.TraceEventSignedAggregateAttestationAndProofElectra:
		return m.handleAggregateAndProofFromAttestationElectra(ctx, clientMeta, event, evt)
	default:
		return fmt.Errorf("unsupported payload type for aggregate and proof: %T", payload)
	}
}

func (m *Mimicry) handleAggregateAndProofFromAttestation(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent,
	payload *eth.TraceEventSignedAggregateAttestationAndProof,
) error {
	if payload.SignedAggregateAttestationAndProof == nil || payload.SignedAggregateAttestationAndProof.GetMessage() == nil {
		return fmt.Errorf("handleAggregateAndProofFromAttestation() called with nil aggregate attestation and proof")
	}

	aggregate := payload.SignedAggregateAttestationAndProof
	message := aggregate.GetMessage()
	attestation := message.GetAggregate()

	if attestation == nil || attestation.GetData() == nil {
		return fmt.Errorf("handleAggregateAndProofFromAttestation() called with nil attestation data")
	}

	attestationData := attestation.GetData()

	// Create the aggregate and proof message using the actual payload data
	aggregateAndProof := &v1.SignedAggregateAttestationAndProof{
		Message: &v1.AggregateAttestationAndProof{
			AggregatorIndex: uint64(message.GetAggregatorIndex()),
			SelectionProof:  fmt.Sprintf("0x%x", message.GetSelectionProof()),
			Aggregate: &v1.Attestation{
				AggregationBits: fmt.Sprintf("0x%x", attestation.GetAggregationBits().Bytes()),
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
				Signature: fmt.Sprintf("0x%x", attestation.GetSignature()),
			},
		},
		Signature: fmt.Sprintf("0x%x", aggregate.GetSignature()),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := m.createAdditionalGossipSubAggregateAndProofData(ctx, event, aggregateAndProof, phase0.Slot(attestationData.GetSlot()), uint64(message.GetAggregatorIndex()), payload.PeerID, payload.Topic, payload.MsgID, payload.MsgSize)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubAggregateAndProof{
		Libp2PTraceGossipsubAggregateAndProof: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubAggregateAndProof{
			Libp2PTraceGossipsubAggregateAndProof: aggregateAndProof,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleAggregateAndProofFromAttestationElectra(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent,
	payload *eth.TraceEventSignedAggregateAttestationAndProofElectra,
) error {
	if payload.SignedAggregateAttestationAndProofElectra == nil || payload.SignedAggregateAttestationAndProofElectra.GetMessage() == nil {
		return fmt.Errorf("handleAggregateAndProofFromAttestationElectra() called with nil aggregate attestation and proof")
	}

	aggregate := payload.SignedAggregateAttestationAndProofElectra
	message := aggregate.GetMessage()
	attestation := message.GetAggregate()

	if attestation == nil || attestation.GetData() == nil {
		return fmt.Errorf("handleAggregateAndProofFromAttestationElectra() called with nil attestation data")
	}

	attestationData := attestation.GetData()

	// Create the aggregate and proof message using the actual payload data
	aggregateAndProof := &v1.SignedAggregateAttestationAndProof{
		Message: &v1.AggregateAttestationAndProof{
			AggregatorIndex: uint64(message.GetAggregatorIndex()),
			SelectionProof:  fmt.Sprintf("0x%x", message.GetSelectionProof()),
			Aggregate: &v1.Attestation{
				AggregationBits: fmt.Sprintf("0x%x", attestation.GetAggregationBits().Bytes()),
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
				Signature: fmt.Sprintf("0x%x", attestation.GetSignature()),
			},
		},
		Signature: fmt.Sprintf("0x%x", aggregate.GetSignature()),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := m.createAdditionalGossipSubAggregateAndProofData(ctx, event, aggregateAndProof, phase0.Slot(attestationData.GetSlot()), uint64(message.GetAggregatorIndex()), payload.PeerID, payload.Topic, payload.MsgID, payload.MsgSize)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubAggregateAndProof{
		Libp2PTraceGossipsubAggregateAndProof: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubAggregateAndProof{
			Libp2PTraceGossipsubAggregateAndProof: aggregateAndProof,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) createAdditionalGossipSubAggregateAndProofData(
	ctx context.Context,
	event *host.TraceEvent,
	aggregateAndProof *v1.SignedAggregateAttestationAndProof,
	slotNumber phase0.Slot,
	aggregatorIndex uint64,
	peerID string,
	topic string,
	msgID string,
	msgSize int,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubAggregateAndProofData, error) {
	wallclockSlot, wallclockEpoch, err := m.ethereum.Metadata().Wallclock().FromTime(event.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	// Add Clock Drift
	timestampAdjusted := event.Timestamp.Add(m.clockDrift)

	aggregateSlot := m.ethereum.Metadata().Wallclock().Slots().FromNumber(uint64(slotNumber))

	// Calculate propagation timing
	diff := timestampAdjusted.Sub(aggregateSlot.TimeWindow().Start()).Milliseconds()

	var propagationSlotStartDiff uint64

	if diff > 0 {
		propagationSlotStartDiff = uint64(diff)
	}

	extra := &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubAggregateAndProofData{
		Peer: &libp2p.Peer{
			Id: peerID,
		},
		WallclockSlot:            timestamppb.New(wallclockSlot.TimeWindow().Start()),
		WallclockEpoch:           timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		PropagationSlotStartDiff: wrapperspb.UInt64(propagationSlotStartDiff),
		AggregatorIndex:          wrapperspb.UInt64(aggregatorIndex),
		SelectionProof:           aggregateAndProof.Message.SelectionProof,
		Topic:                    wrapperspb.String(topic),
		MessageSize:              wrapperspb.UInt32(uint32(msgSize)), //nolint:gosec // int -> uint32 common conversion pattern in xatu.
		MessageId:                wrapperspb.String(msgID),
	}

	return extra, nil
}
