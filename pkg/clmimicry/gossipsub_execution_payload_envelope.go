package clmimicry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (p *Processor) handleGossipExecutionPayloadEnvelope(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventExecutionPayloadEnvelope,
) error {
	if payload.ExecutionPayloadEnvelope == nil {
		return fmt.Errorf("handleGossipExecutionPayloadEnvelope() called with nil envelope")
	}

	envelope := payload.ExecutionPayloadEnvelope.GetMessage()
	if envelope == nil {
		return fmt.Errorf("handleGossipExecutionPayloadEnvelope() called with nil envelope message")
	}

	executionPayload := envelope.GetPayload()

	data := &gossipsub.ExecutionPayloadEnvelope{
		Slot:            wrapperspb.UInt64(uint64(executionPayload.GetSlotNumber())),
		BuilderIndex:    wrapperspb.UInt64(uint64(envelope.GetBuilderIndex())),
		BeaconBlockRoot: wrapperspb.String(fmt.Sprintf("0x%x", envelope.GetBeaconBlockRoot())),
		BlockHash:       wrapperspb.String(fmt.Sprintf("0x%x", executionPayload.GetBlockHash())),
		StateRoot:       wrapperspb.String(fmt.Sprintf("0x%x", executionPayload.GetStateRoot())),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubExecutionPayloadEnvelopeData(payload, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubExecutionPayloadEnvelope{
		Libp2PTraceGossipsubExecutionPayloadEnvelope: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_ENVELOPE,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubExecutionPayloadEnvelope{
			Libp2PTraceGossipsubExecutionPayloadEnvelope: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (p *Processor) createAdditionalGossipSubExecutionPayloadEnvelopeData(
	payload *TraceEventExecutionPayloadEnvelope,
	timestamp time.Time,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubExecutionPayloadEnvelopeData, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	slotNumber := uint64(payload.ExecutionPayloadEnvelope.GetMessage().GetPayload().GetSlotNumber())
	slot := p.wallclock.Slots().FromNumber(slotNumber)
	epoch := p.wallclock.Epochs().FromSlot(slotNumber)
	timestampAdjusted := timestamp.Add(p.clockDrift)

	return &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubExecutionPayloadEnvelopeData{
		WallclockSlot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(wallclockSlot.Number()),
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(wallclockEpoch.Number()),
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
		Epoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(epoch.Number()),
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
		Slot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(slotNumber),
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		},
		Propagation: &xatu.PropagationV2{
			SlotStartDiff: &wrapperspb.UInt64Value{
				Value: uint64(timestampAdjusted.Sub(slot.TimeWindow().Start()).Milliseconds()),
			},
		},
		Metadata:    &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(payload.PeerID)},
		Topic:       wrapperspb.String(payload.Topic),
		MessageSize: wrapperspb.UInt32(uint32(payload.MsgSize)),
		MessageId:   wrapperspb.String(payload.MsgID),
	}, nil
}
