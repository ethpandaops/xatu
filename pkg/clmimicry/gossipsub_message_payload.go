package clmimicry

import (
	"context"
	"errors"
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

// Validation outcomes for a gossipsub message payload event.
const (
	MessagePayloadOutcomeDeliver = "deliver"
	MessagePayloadOutcomeReject  = "reject"
)

// handleGossipsubMessagePayloadEvent routes a GOSSIPSUB_MESSAGE_PAYLOAD trace
// event through the enabled/metrics/sharding gates shared by all gossipsub events.
func (p *Processor) handleGossipsubMessagePayloadEvent(
	ctx context.Context,
	event *TraceEvent,
	clientMeta *xatu.ClientMeta,
) error {
	if !p.events.GossipSubMessagePayloadEnabled {
		return nil
	}

	xatuEvent := xatu.Event_LIBP2P_TRACE_GOSSIPSUB_MESSAGE_PAYLOAD.String()

	networkStr := getNetworkID(clientMeta)
	p.metrics.AddEvent(xatuEvent, networkStr)

	if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
		return nil
	}

	payload, ok := event.Payload.(*TraceEventMessagePayload)
	if !ok {
		return errors.New("invalid payload type for GOSSIPSUB_MESSAGE_PAYLOAD event")
	}

	return p.handleGossipMessagePayload(ctx, clientMeta, event, payload)
}

func (p *Processor) handleGossipMessagePayload(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventMessagePayload,
) error {
	if payload.Data == nil {
		return fmt.Errorf("handleGossipMessagePayload() called with nil message data")
	}

	data := &gossipsub.MessagePayload{
		Data:         payload.Data,
		Outcome:      wrapperspb.String(payload.Outcome),
		RejectReason: wrapperspb.String(payload.RejectReason),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubMessagePayloadData(payload, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubMessagePayload{
		Libp2PTraceGossipsubMessagePayload: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_MESSAGE_PAYLOAD,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubMessagePayload{
			Libp2PTraceGossipsubMessagePayload: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (p *Processor) createAdditionalGossipSubMessagePayloadData(
	payload *TraceEventMessagePayload,
	timestamp time.Time,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubMessagePayloadData, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	return &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubMessagePayloadData{
		WallclockSlot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(wallclockSlot.Number()),
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(wallclockEpoch.Number()),
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
		Metadata:    &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(payload.PeerID)},
		Topic:       wrapperspb.String(payload.Topic),
		MessageSize: wrapperspb.UInt32(uint32(payload.MsgSize)),
		MessageId:   wrapperspb.String(payload.MsgID),
	}, nil
}
