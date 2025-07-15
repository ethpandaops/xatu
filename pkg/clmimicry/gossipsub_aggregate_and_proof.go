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

// deriveCommitteeIndexFromBits finds the committee index from committee bits
// For Electra aggregates, only one committee bit should be set
func deriveCommitteeIndexFromBits(committeeBits []byte) uint64 {
	for i := 0; i < len(committeeBits)*8; i++ {
		byteIndex := i / 8
		bitIndex := i % 8

		if byteIndex < len(committeeBits) && (committeeBits[byteIndex]&(1<<bitIndex)) != 0 {
			return uint64(i) //nolint:gosec // i is always in valid range for uint64
		}
	}

	return 0 // Default fallback if no bit is set
}

func (p *Processor) handleGossipAggregateAndProof(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent,
	payload any,
) error {
	switch evt := payload.(type) {
	case *eth.TraceEventSignedAggregateAttestationAndProof:
		return p.handleAggregateAndProofFromAttestation(ctx, clientMeta, event, evt)
	case *eth.TraceEventSignedAggregateAttestationAndProofElectra:
		return p.handleAggregateAndProofFromAttestationElectra(ctx, clientMeta, event, evt)
	default:
		return fmt.Errorf("unsupported payload type for aggregate and proof: %T", payload)
	}
}

func (p *Processor) handleAggregateAndProofFromAttestation(
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

	// Create the aggregate and proof message using the actual payload data with V2 types
	aggregateAndProof := &v1.SignedAggregateAttestationAndProofV2{
		Message: &v1.AggregateAttestationAndProofV2{
			AggregatorIndex: wrapperspb.UInt64(uint64(message.GetAggregatorIndex())),
			Aggregate: &v1.AttestationV2{
				AggregationBits: fmt.Sprintf("0x%x", attestation.GetAggregationBits().Bytes()),
				Data: &v1.AttestationDataV2{
					Slot:            wrapperspb.UInt64(uint64(attestationData.GetSlot())),
					BeaconBlockRoot: fmt.Sprintf("0x%x", attestationData.GetBeaconBlockRoot()),
					Source: &v1.CheckpointV2{
						Epoch: wrapperspb.UInt64(uint64(attestationData.GetSource().GetEpoch())),
						Root:  fmt.Sprintf("0x%x", attestationData.GetSource().GetRoot()),
					},
					Target: &v1.CheckpointV2{
						Epoch: wrapperspb.UInt64(uint64(attestationData.GetTarget().GetEpoch())),
						Root:  fmt.Sprintf("0x%x", attestationData.GetTarget().GetRoot()),
					},
					Index: wrapperspb.UInt64(uint64(attestationData.GetCommitteeIndex())),
				},
				Signature: fmt.Sprintf("0x%x", attestation.GetSignature()),
			},
		},
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubAggregateAndProofData(ctx, event, aggregateAndProof, phase0.Slot(attestationData.GetSlot()), uint64(message.GetAggregatorIndex()), payload.PeerID, payload.Topic, payload.MsgID, payload.MsgSize)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubAggregateAndProof{
		Libp2PTraceGossipsubAggregateAndProof: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubAggregateAndProof{
			Libp2PTraceGossipsubAggregateAndProof: aggregateAndProof,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleAggregateAndProofFromAttestationElectra(
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

	// Create the aggregate and proof message using the actual payload data with V2 types
	aggregateAndProof := &v1.SignedAggregateAttestationAndProofV2{
		Message: &v1.AggregateAttestationAndProofV2{
			AggregatorIndex: wrapperspb.UInt64(uint64(message.GetAggregatorIndex())),
			Aggregate: &v1.AttestationV2{
				AggregationBits: fmt.Sprintf("0x%x", attestation.GetAggregationBits().Bytes()),
				Data: &v1.AttestationDataV2{
					Slot:            wrapperspb.UInt64(uint64(attestationData.GetSlot())),
					BeaconBlockRoot: fmt.Sprintf("0x%x", attestationData.GetBeaconBlockRoot()),
					Source: &v1.CheckpointV2{
						Epoch: wrapperspb.UInt64(uint64(attestationData.GetSource().GetEpoch())),
						Root:  fmt.Sprintf("0x%x", attestationData.GetSource().GetRoot()),
					},
					Target: &v1.CheckpointV2{
						Epoch: wrapperspb.UInt64(uint64(attestationData.GetTarget().GetEpoch())),
						Root:  fmt.Sprintf("0x%x", attestationData.GetTarget().GetRoot()),
					},
					// For Electra, derive committee index from committee bits
					Index: wrapperspb.UInt64(deriveCommitteeIndexFromBits(attestation.GetCommitteeBits())),
				},
				Signature: fmt.Sprintf("0x%x", attestation.GetSignature()),
			},
		},
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubAggregateAndProofData(ctx, event, aggregateAndProof, phase0.Slot(attestationData.GetSlot()), uint64(message.GetAggregatorIndex()), payload.PeerID, payload.Topic, payload.MsgID, payload.MsgSize)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubAggregateAndProof{
		Libp2PTraceGossipsubAggregateAndProof: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubAggregateAndProof{
			Libp2PTraceGossipsubAggregateAndProof: aggregateAndProof,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) createAdditionalGossipSubAggregateAndProofData(
	ctx context.Context,
	event *host.TraceEvent,
	aggregateAndProof *v1.SignedAggregateAttestationAndProofV2,
	slotNumber phase0.Slot,
	aggregatorIndex uint64,
	peerID string,
	topic string,
	msgID string,
	msgSize int,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubAggregateAndProofData, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(event.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	slot := p.wallclock.Slots().FromNumber(uint64(slotNumber))
	epoch := p.wallclock.Epochs().FromSlot(uint64(slotNumber))
	timestampAdjusted := event.Timestamp.Add(p.clockDrift)

	// Calculate propagation timing
	diff := timestampAdjusted.Sub(slot.TimeWindow().Start()).Milliseconds()

	var propagationSlotStartDiff uint64

	if diff > 0 {
		propagationSlotStartDiff = uint64(diff)
	}

	extra := &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubAggregateAndProofData{
		Epoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(epoch.Number()),
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
		Slot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(uint64(slotNumber)),
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(wallclockEpoch.Number()),
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
		WallclockSlot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(wallclockSlot.Number()),
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		Propagation: &xatu.PropagationV2{
			SlotStartDiff: wrapperspb.UInt64(propagationSlotStartDiff),
		},
		AggregatorIndex: wrapperspb.UInt64(aggregatorIndex),
		Metadata:        &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(peerID)},
		Topic:           wrapperspb.String(topic),
		MessageSize:     wrapperspb.UInt32(uint32(msgSize)), //nolint:gosec // int -> uint32 common conversion pattern in xatu.
		MessageId:       wrapperspb.String(msgID),
	}

	return extra, nil
}
