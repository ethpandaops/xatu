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

	// Cast the event type to our xatu Libp2p event types
	typ := libp2p.EventTypeFromHermesEventType(event.Type)
	if typ == libp2p.EventType_UNKNOWN {
		return errors.Errorf("unknown event type: %v", event.Type)
	}

	clientMeta, err := m.createNewClientMeta(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to create new client meta")
	}

	traceMeta := &libp2p.TraceEventMetadata{
		PeerId:               wrapperspb.String(event.PeerID.String()),
		SessionStartDateTime: timestamppb.New(m.startupTime),
	}

	switch typ {
	case libp2p.EventType_ADD_PEER:
		return m.handleAddPeerEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_PUBLISH_MESSAGE:
		return m.handlePublishMessageEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_REJECT_MESSAGE:
		return m.handleRejectMessageEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_DUPLICATE_MESSAGE:
		return m.handleDuplicateMessageEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_DELIVER_MESSAGE:
		return m.handleDeliverMessageEvent(ctx, clientMeta, traceMeta, event)
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
	case libp2p.EventType_DROP_RPC:
		return m.handleDropRPCEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_JOIN:
		return m.handleJoinEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_LEAVE:
		return m.handleLeaveEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_GRAFT:
		return m.handleGraftEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_PRUNE:
		return m.handlePruneEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_VALIDATE_MESSAGE:
		return m.handleValidateMessageEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_THROTTLE_PEER:
		return m.handleThrottlePeerEvent(ctx, clientMeta, traceMeta, event)
	case libp2p.EventType_UNDELIVERABLE_MESSAGE:
		return m.handleUndeliverableMessageEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_STREAM:
	// 	return m.handleHandleStreamEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_STATUS:
	// 	return m.handleHandleStatusEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_METADATA:
	// 	return m.handleHandleMetadataEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_BLOB_SIDECARS_BY_RANGE:
	// 	return m.handleHandleBlobSidecarsByRangeEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_BLOB_SIDECARS_BY_ROOT:
	// 	return m.handleHandleBlobSidecarsByRootEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_PING:
	// 	return m.handleHandlePingEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_GOODBYE:
	// 	return m.handleHandleGoodbyeEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_BEACON_BLOCKS_BY_RANGE:
	// 	return m.handleHandleBeaconBlocksByRangeEvent(ctx, clientMeta, traceMeta, event)
	// case libp2p.EventType_HANDLE_BEACON_BLOCKS_BY_ROOT:
	// 	return m.handleHandleBeaconBlocksByRootEvent(ctx, clientMeta, traceMeta, event)
	default:
		return errors.Errorf("unsupported event type: %v", typ)
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

func (m *Mimicry) handlePublishMessageEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToPublishMessage(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to publish message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTracePublishMessage{
		Libp2PTracePublishMessage: &xatu.ClientMeta_AdditionalLibP2PTracePublishMessageData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTracePublishMessage{
			Libp2PTracePublishMessage: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleRejectMessageEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToRejectMessage(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to reject message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRejectMessage{
		Libp2PTraceRejectMessage: &xatu.ClientMeta_AdditionalLibP2PTraceRejectMessageData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceRejectMessage{
			Libp2PTraceRejectMessage: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleDuplicateMessageEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToDuplicateMessage(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to duplicate message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceDuplicateMessage{
		Libp2PTraceDuplicateMessage: &xatu.ClientMeta_AdditionalLibP2PTraceDuplicateMessageData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceDuplicateMessage{
			Libp2PTraceDuplicateMessage: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleDeliverMessageEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToDeliverMessage(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to deliver message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceDeliverMessage{
		Libp2PTraceDeliverMessage: &xatu.ClientMeta_AdditionalLibP2PTraceDeliverMessageData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceDeliverMessage{
			Libp2PTraceDeliverMessage: data,
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

func (m *Mimicry) handleDropRPCEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToDropRPC(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to drop rpc event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceDropRpc{
		Libp2PTraceDropRpc: &xatu.ClientMeta_AdditionalLibP2PTraceDropRPCData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_DROP_RPC,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceDropRpc{
			Libp2PTraceDropRpc: data,
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

func (m *Mimicry) handleLeaveEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToLeave(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to leave event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceLeave{
		Libp2PTraceLeave: &xatu.ClientMeta_AdditionalLibP2PTraceLeaveData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_LEAVE,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceLeave{
			Libp2PTraceLeave: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleGraftEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToGraft(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to graft event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGraft{
		Libp2PTraceGraft: &xatu.ClientMeta_AdditionalLibP2PTraceGraftData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GRAFT,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGraft{
			Libp2PTraceGraft: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handlePruneEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToPrune(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to prune event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTracePrune{
		Libp2PTracePrune: &xatu.ClientMeta_AdditionalLibP2PTracePruneData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_PRUNE,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTracePrune{
			Libp2PTracePrune: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleValidateMessageEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToValidateMessage(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to validate message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceValidateMessage{
		Libp2PTraceValidateMessage: &xatu.ClientMeta_AdditionalLibP2PTraceValidateMessageData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_VALIDATE_MESSAGE,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceValidateMessage{
			Libp2PTraceValidateMessage: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleThrottlePeerEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToThrottlePeer(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to throttle peer event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceThrottlePeer{
		Libp2PTraceThrottlePeer: &xatu.ClientMeta_AdditionalLibP2PTraceThrottlePeerData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_THROTTLE_PEER,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceThrottlePeer{
			Libp2PTraceThrottlePeer: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) handleUndeliverableMessageEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent) error {
	data, err := libp2p.TraceEventToUndeliverableMessage(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to undeliverable message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceUndeliverableMessage{
		Libp2PTraceUndeliverableMessage: &xatu.ClientMeta_AdditionalLibP2PTraceUndeliverableMessageData{
			Metadata: traceMeta,
		},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_UNDELIVERABLE_MESSAGE,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceUndeliverableMessage{
			Libp2PTraceUndeliverableMessage: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}
