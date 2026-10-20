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

// handlePartialDataColumnEvent applies the enabled/metrics/sharding gates shared by all gossipsub events.
func (p *Processor) handlePartialDataColumnEvent(
	ctx context.Context,
	event *TraceEvent,
	clientMeta *xatu.ClientMeta,
) error {
	switch event.Type {
	case TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN:
		if !p.shouldEmitPartialEvent(p.events.GossipSubPartialDataColumnEnabled, event, clientMeta, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN) {
			return nil
		}

		payload, ok := event.Payload.(*TraceEventPartialDataColumn)
		if !ok {
			return errors.New("invalid payload type for GOSSIPSUB_PARTIAL_DATA_COLUMN event")
		}

		return p.handlePartialDataColumn(ctx, clientMeta, event, payload)
	case TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME:
		if !p.shouldEmitPartialEvent(p.events.GossipSubPartialDataColumnOutcomeEnabled, event, clientMeta, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME) {
			return nil
		}

		payload, ok := event.Payload.(*TraceEventPartialDataColumnOutcome)
		if !ok {
			return errors.New("invalid payload type for GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME event")
		}

		return p.handlePartialDataColumnOutcome(ctx, clientMeta, event, payload)
	case TraceEvent_DATA_COLUMN_COMPLETED:
		if !p.shouldEmitPartialEvent(p.events.DataColumnCompletedEnabled, event, clientMeta, xatu.Event_LIBP2P_TRACE_DATA_COLUMN_COMPLETED) {
			return nil
		}

		payload, ok := event.Payload.(*TraceEventDataColumnCompleted)
		if !ok {
			return errors.New("invalid payload type for DATA_COLUMN_COMPLETED event")
		}

		return p.handleDataColumnCompleted(ctx, clientMeta, event, payload)
	default:
		return fmt.Errorf("unsupported partial data column event type %q", event.Type)
	}
}

func (p *Processor) shouldEmitPartialEvent(
	enabled bool,
	event *TraceEvent,
	clientMeta *xatu.ClientMeta,
	name xatu.Event_Name,
) bool {
	if !enabled {
		return false
	}

	p.metrics.AddEvent(name.String(), getNetworkID(clientMeta))

	return p.ShouldTraceMessage(event, clientMeta, name.String())
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (p *Processor) handlePartialDataColumn(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventPartialDataColumn,
) error {
	wallclockSlot, wallclockEpoch, err := p.partialWallclock(event.Timestamp)
	if err != nil {
		return err
	}

	slot, epoch, propagation := p.partialBlockSlot(payload.Slot, event.Timestamp)

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return errors.New("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubPartialDataColumn{
		Libp2PTraceGossipsubPartialDataColumn: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubPartialDataColumnData{
			Epoch:          epoch,
			Slot:           slot,
			WallclockEpoch: wallclockEpoch,
			WallclockSlot:  wallclockSlot,
			Propagation:    propagation,
			Metadata:       &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(payload.PeerID)},
			Topic:          wrapperspb.String(partialTopic(event, payload.Topic)),
			MessageSize:    wrapperspb.UInt32(uint32(payload.MsgSize)),
		},
	}

	decorated := p.partialDecoratedEvent(xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN, event, metadata)
	decorated.Data = &xatu.DecoratedEvent_Libp2PTraceGossipsubPartialDataColumn{
		Libp2PTraceGossipsubPartialDataColumn: &gossipsub.PartialDataColumn{
			Direction:              wrapperspb.String(payload.Direction),
			CapturePoint:           wrapperspb.String(payload.CapturePoint),
			GroupId:                wrapperspb.String(payload.GroupID),
			GroupVersion:           wrapperspb.UInt32(payload.GroupVersion),
			BlockRoot:              wrapperspb.String(payload.BlockRoot),
			ColumnIndex:            wrapperspb.UInt64(payload.ColumnIndex),
			HasMessage:             wrapperspb.Bool(payload.HasMessage),
			CellsCount:             wrapperspb.UInt32(payload.CellsCount),
			CellsBitmap:            wrapperspb.String(payload.CellsBitmap),
			BitmapLength:           wrapperspb.UInt32(payload.BitmapLength),
			HasHeader:              wrapperspb.Bool(payload.HasHeader),
			KzgCommitmentsCount:    wrapperspb.UInt32(payload.KZGCommitmentsCount),
			ProposerIndex:          wrapperspb.UInt64(payload.ProposerIndex),
			HasMetadata:            wrapperspb.Bool(payload.HasMetadata),
			MetadataSize:           wrapperspb.UInt32(payload.MetadataSize),
			MetadataAvailableCount: wrapperspb.UInt32(payload.MetadataAvailableCount),
			MetadataRequestsCount:  wrapperspb.UInt32(payload.MetadataRequestsCount),
			MetadataAvailable:      wrapperspb.String(payload.MetadataAvailable),
			MetadataRequests:       wrapperspb.String(payload.MetadataRequests),
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decorated)
}

func (p *Processor) handlePartialDataColumnOutcome(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventPartialDataColumnOutcome,
) error {
	wallclockSlot, wallclockEpoch, err := p.partialWallclock(event.Timestamp)
	if err != nil {
		return err
	}

	slot, epoch, _ := p.partialBlockSlot(payload.Slot, event.Timestamp)

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return errors.New("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubPartialDataColumnOutcome{
		Libp2PTraceGossipsubPartialDataColumnOutcome: &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubPartialDataColumnOutcomeData{
			Epoch:          epoch,
			Slot:           slot,
			WallclockEpoch: wallclockEpoch,
			WallclockSlot:  wallclockSlot,
			Metadata:       &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(payload.PeerID)},
			Topic:          wrapperspb.String(partialTopic(event, payload.Topic)),
		},
	}

	decorated := p.partialDecoratedEvent(xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME, event, metadata)
	decorated.Data = &xatu.DecoratedEvent_Libp2PTraceGossipsubPartialDataColumnOutcome{
		Libp2PTraceGossipsubPartialDataColumnOutcome: &gossipsub.PartialDataColumnOutcome{
			GroupId:       wrapperspb.String(payload.GroupID),
			BlockRoot:     wrapperspb.String(payload.BlockRoot),
			ColumnIndex:   wrapperspb.UInt64(payload.ColumnIndex),
			Stage:         wrapperspb.String(payload.Stage),
			Outcome:       wrapperspb.String(payload.Outcome),
			Reason:        wrapperspb.String(payload.Reason),
			CellsCount:    wrapperspb.UInt32(payload.CellsCount),
			NewCellsCount: wrapperspb.UInt32(payload.NewCellsCount),
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decorated)
}

func (p *Processor) handleDataColumnCompleted(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventDataColumnCompleted,
) error {
	wallclockSlot, wallclockEpoch, err := p.partialWallclock(event.Timestamp)
	if err != nil {
		return err
	}

	slot, epoch, propagation := p.partialBlockSlot(payload.Slot, event.Timestamp)

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return errors.New("failed to clone client metadata")
	}

	additional := &xatu.ClientMeta_AdditionalLibP2PTraceDataColumnCompletedData{
		Epoch:          epoch,
		Slot:           slot,
		WallclockEpoch: wallclockEpoch,
		WallclockSlot:  wallclockSlot,
		Propagation:    propagation,
		Topic:          wrapperspb.String(partialTopic(event, payload.Topic)),
	}

	// Columns built from the EL or by reconstruction have no completing peer.
	if payload.PeerID != "" {
		additional.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(payload.PeerID)}
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceDataColumnCompleted{
		Libp2PTraceDataColumnCompleted: additional,
	}

	decorated := p.partialDecoratedEvent(xatu.Event_LIBP2P_TRACE_DATA_COLUMN_COMPLETED, event, metadata)
	decorated.Data = &xatu.DecoratedEvent_Libp2PTraceDataColumnCompleted{
		Libp2PTraceDataColumnCompleted: &gossipsub.DataColumnCompleted{
			BlockRoot:           wrapperspb.String(payload.BlockRoot),
			ColumnIndex:         wrapperspb.UInt64(payload.ColumnIndex),
			CompletedVia:        wrapperspb.String(payload.CompletedVia),
			CellsTotal:          wrapperspb.UInt32(payload.CellsTotal),
			CellsFromGossip:     wrapperspb.UInt32(payload.CellsFromGossip),
			CellsFromEl:         wrapperspb.UInt32(payload.CellsFromEL),
			ContributingPeers:   wrapperspb.UInt32(payload.ContributingPeers),
			KzgCommitmentsCount: wrapperspb.UInt32(payload.KZGCommitmentsCount),
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decorated)
}

// partialDecoratedEvent builds the event envelope; callers set Data.
func (p *Processor) partialDecoratedEvent(
	name xatu.Event_Name,
	event *TraceEvent,
	metadata *xatu.ClientMeta,
) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     name,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
	}
}

func (p *Processor) partialWallclock(timestamp time.Time) (*xatu.SlotV2, *xatu.EpochV2, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(timestamp)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	return &xatu.SlotV2{
			Number:        wrapperspb.UInt64(wallclockSlot.Number()),
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		}, &xatu.EpochV2{
			Number:        wrapperspb.UInt64(wallclockEpoch.Number()),
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		}, nil
}

// partialBlockSlot returns nil slot, epoch and propagation when the block slot is unknown (zero).
//
//nolint:gosec // int64 -> uint64 common conversion pattern in xatu.
func (p *Processor) partialBlockSlot(
	slotNumber uint64,
	timestamp time.Time,
) (*xatu.SlotV2, *xatu.EpochV2, *xatu.PropagationV2) {
	if slotNumber == 0 {
		return nil, nil, nil
	}

	slot := p.wallclock.Slots().FromNumber(slotNumber)
	epoch := p.wallclock.Epochs().FromSlot(slotNumber)

	return &xatu.SlotV2{
			Number:        wrapperspb.UInt64(slotNumber),
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		}, &xatu.EpochV2{
			Number:        wrapperspb.UInt64(epoch.Number()),
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		}, &xatu.PropagationV2{
			SlotStartDiff: wrapperspb.UInt64(
				uint64(timestamp.Add(p.clockDrift).Sub(slot.TimeWindow().Start()).Milliseconds()),
			),
		}
}

// partialTopic prefers the payload topic and falls back to the trace event topic.
func partialTopic(event *TraceEvent, payloadTopic string) string {
	if payloadTopic != "" {
		return payloadTopic
	}

	return event.Topic
}
