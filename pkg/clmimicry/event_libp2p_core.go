package clmimicry

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (p *Processor) handleConnectedEvent(ctx context.Context,
	event *ConnectedEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.ConnectedEnabled {
		return nil
	}

	// Build protobuf message directly from typed event
	data := &libp2p.Connected{
		RemotePeer:   wrapperspb.String(event.RemotePeer),
		RemoteMaddrs: wrapperspb.String(event.RemoteMaddrs.String()),
		AgentVersion: wrapperspb.String(event.AgentVersion),
		Direction:    wrapperspb.String(event.Direction),
		Opened:       timestamppb.New(event.Opened),
		Limited:      wrapperspb.Bool(event.Limited),
		Transient:    wrapperspb.Bool(event.Limited),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceConnected{
		Libp2PTraceConnected: &xatu.ClientMeta_AdditionalLibP2PTraceConnectedData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_CONNECTED,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceConnected{
			Libp2PTraceConnected: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleDisconnectedEvent(ctx context.Context,
	event *DisconnectedEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.DisconnectedEnabled {
		return nil
	}

	// Build protobuf message directly from typed event
	data := &libp2p.Disconnected{
		RemotePeer:   wrapperspb.String(event.RemotePeer),
		RemoteMaddrs: wrapperspb.String(event.RemoteMaddrs.String()),
		AgentVersion: wrapperspb.String(event.AgentVersion),
		Direction:    wrapperspb.String(event.Direction),
		Opened:       timestamppb.New(event.Opened),
		Limited:      wrapperspb.Bool(event.Limited),
		Transient:    wrapperspb.Bool(event.Limited),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceDisconnected{
		Libp2PTraceDisconnected: &xatu.ClientMeta_AdditionalLibP2PTraceDisconnectedData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_DISCONNECTED,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceDisconnected{
			Libp2PTraceDisconnected: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleSyntheticHeartbeatEvent(ctx context.Context,
	event *SyntheticHeartbeatEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.SyntheticHeartbeatEnabled {
		return nil
	}

	// Build protobuf message directly from typed event
	data := &libp2p.SyntheticHeartbeat{
		Timestamp:       timestamppb.New(event.Timestamp),
		RemotePeer:      wrapperspb.String(event.RemotePeer),
		RemoteMaddrs:    wrapperspb.String(event.RemoteMaddrs.String()),
		LatencyMs:       wrapperspb.Int64(event.LatencyMs),
		AgentVersion:    wrapperspb.String(event.AgentVersion),
		Direction:       wrapperspb.UInt32(event.Direction),
		Protocols:       event.Protocols,
		ConnectionAgeNs: wrapperspb.Int64(event.ConnectionAgeNs),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceSyntheticHeartbeat{
		Libp2PTraceSyntheticHeartbeat: &xatu.ClientMeta_AdditionalLibP2PTraceSyntheticHeartbeatData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceSyntheticHeartbeat{
			Libp2PTraceSyntheticHeartbeat: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}
