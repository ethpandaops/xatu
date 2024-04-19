package clmimicry

import (
	"context"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (m *Mimicry) handleHermesEvent(ctx context.Context, event *host.TraceEvent) error {
	if event == nil {
		return errors.New("event is nil")
	}

	// Log the event type
	m.log.WithField("type", event.Type).Debug("Received Hermes event")

	// Cast the event type to our xatu Libp2p event types
	typ := libp2p.EventTypeFromHermesEventType(event.Type)
	if typ == libp2p.EventType_UNKNOWN {
		// Log an error and return nil if the event type is not supported
		m.log.Debugf("Got unsupported Hermes event type: %v", event.Type)

		return nil
	}

	clientMeta, err := m.createNewClientMeta(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to create new client meta")
	}

	traceMeta := &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String(event.PeerID.String()),
	}

	switch typ {
	case libp2p.EventType_ADD_PEER:
		return m.handleAddPeerEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_RECV_RPC:
		return m.handleRecvRPCEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_SEND_RPC:
		return m.handleSendRPCEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_CONNECTED:
		return m.handleConnectedEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_DISCONNECTED:
		return m.handleDisconnectedEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_REMOVE_PEER:
		return m.handleRemovePeerEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_JOIN:
		return m.handleJoinEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_HANDLE_METADATA:
		return m.handleHandleMetadataEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_HANDLE_STATUS:
		return m.handleHandleStatusEvent(ctx, clientMeta, traceMeta, event)
	default:
		return fmt.Errorf("unsupported event type: %v", typ)
	}
}

func (m *Mimicry) handleAddPeerEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToAddPeer(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to add_peer event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceAddPeer{
		Libp2PTraceAddPeer: &xatu.ClientMeta_AdditionalLibP2PTraceAddPeerData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_ADD_PEER,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceAddPeer{
			Libp2PTraceAddPeer: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleSendRPCEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToSendRPC(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to deliver message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceSendRpc{
		Libp2PTraceSendRpc: &xatu.ClientMeta_AdditionalLibP2PTraceSendRPCData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_SEND_RPC,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceSendRpc{
			Libp2PTraceSendRpc: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleRecvRPCEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToRecvRPC(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to deliver message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRecvRpc{
		Libp2PTraceRecvRpc: &xatu.ClientMeta_AdditionalLibP2PTraceRecvRPCData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RECV_RPC,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceRecvRpc{
			Libp2PTraceRecvRpc: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleConnectedEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToConnected(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to connected event")
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
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceConnected{
			Libp2PTraceConnected: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleDisconnectedEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToDisconnected(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to disconnected event")
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
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceDisconnected{
			Libp2PTraceDisconnected: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleRemovePeerEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToRemovePeer(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to remove peer event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRemovePeer{
		Libp2PTraceRemovePeer: &xatu.ClientMeta_AdditionalLibP2PTraceRemovePeerData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceRemovePeer{
			Libp2PTraceRemovePeer: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleJoinEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToJoin(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to join event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceJoin{
		Libp2PTraceJoin: &xatu.ClientMeta_AdditionalLibP2PTraceJoinData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_JOIN,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceJoin{
			Libp2PTraceJoin: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleHandleMetadataEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToHandleMetadata(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to handle metadata event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceHandleMetadata{
		Libp2PTraceHandleMetadata: &xatu.ClientMeta_AdditionalLibP2PTraceHandleMetadataData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_HANDLE_METADATA,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceHandleMetadata{
			Libp2PTraceHandleMetadata: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleHandleStatusEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToHandleStatus(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to handle status event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceHandleStatus{
		Libp2PTraceHandleStatus: &xatu.ClientMeta_AdditionalLibP2PTraceHandleStatusData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_HANDLE_STATUS,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceHandleStatus{
			Libp2PTraceHandleStatus: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}
