package clmimicry

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/pkg/errors"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Map of libp2p event types to Xatu event types.
// This serves both for mapping and for checking if an event is a libp2p event.
var libp2pToXatuEventMap = map[string]string{
	pubsubpb.TraceEvent_PUBLISH_MESSAGE.String():   xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE.String(),
	pubsubpb.TraceEvent_REJECT_MESSAGE.String():    xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE.String(),
	pubsubpb.TraceEvent_DUPLICATE_MESSAGE.String(): xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String(),
	pubsubpb.TraceEvent_DELIVER_MESSAGE.String():   xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE.String(),
	pubsubpb.TraceEvent_ADD_PEER.String():          xatu.Event_LIBP2P_TRACE_ADD_PEER.String(),
	pubsubpb.TraceEvent_REMOVE_PEER.String():       xatu.Event_LIBP2P_TRACE_REMOVE_PEER.String(),
	pubsubpb.TraceEvent_RECV_RPC.String():          xatu.Event_LIBP2P_TRACE_RECV_RPC.String(),
	pubsubpb.TraceEvent_SEND_RPC.String():          xatu.Event_LIBP2P_TRACE_SEND_RPC.String(),
	pubsubpb.TraceEvent_DROP_RPC.String():          xatu.Event_LIBP2P_TRACE_DROP_RPC.String(),
	pubsubpb.TraceEvent_JOIN.String():              xatu.Event_LIBP2P_TRACE_JOIN.String(),
	pubsubpb.TraceEvent_LEAVE.String():             xatu.Event_LIBP2P_TRACE_LEAVE.String(),
	pubsubpb.TraceEvent_GRAFT.String():             xatu.Event_LIBP2P_TRACE_GRAFT.String(),
	pubsubpb.TraceEvent_PRUNE.String():             xatu.Event_LIBP2P_TRACE_PRUNE.String(),
}

// handleHermesLibp2pEvent handles libp2p pubsub protocol level events.
func (m *Mimicry) handleHermesLibp2pEvent(
	ctx context.Context,
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	// Map libp2p event to Xatu event.
	xatuEvent, err := mapLibp2pEventToXatuEvent(event.Type)
	if err != nil {
		m.log.WithField("event", event.Type).Tracef("unsupported event in handleHermesLibp2pEvent event")

		//nolint:nilerr // we don't want to return an error here.
		return nil
	}

	switch xatuEvent {
	case xatu.Event_LIBP2P_TRACE_ADD_PEER.String():
		if !m.Config.Events.AddPeerEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleAddPeerEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_RECV_RPC.String():
		if !m.Config.Events.RecvRPCEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleRecvRPCEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_DROP_RPC.String():
		if !m.Config.Events.DropRPCEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleDropRPCEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_SEND_RPC.String():
		if !m.Config.Events.SendRPCEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleSendRPCEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_REMOVE_PEER.String():
		if !m.Config.Events.RemovePeerEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleRemovePeerEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_JOIN.String():
		if !m.Config.Events.JoinEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleJoinEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_LEAVE.String():
		if !m.Config.Events.LeaveEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleLeaveEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_GRAFT.String():
		if !m.Config.Events.GraftEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleGraftEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_PRUNE.String():
		if !m.Config.Events.PruneEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handlePruneEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE.String():
		if !m.Config.Events.PublishMessageEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handlePublishMessageEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE.String():
		if !m.Config.Events.RejectMessageEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleRejectMessageEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String():
		if !m.Config.Events.DuplicateMessageEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleDuplicateMessageEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE.String():
		if !m.Config.Events.DeliverMessageEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config..
		if !m.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return m.handleDeliverMessageEvent(ctx, clientMeta, traceMeta, event)
	}

	return nil
}

func (m *Mimicry) handleRemovePeerEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleJoinEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleLeaveEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleGraftEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handlePruneEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleSendRPCEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleAddPeerEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleRecvRPCEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
	var decoratedEvents []*xatu.DecoratedEvent

	data, err := libp2p.TraceEventToRecvRPC(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to deliver message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	// 1. Root level rpc event.
	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRecvRpc{
		Libp2PTraceRecvRpc: &xatu.ClientMeta_AdditionalLibP2PTraceRecvRPCData{
			Metadata: traceMeta,
		},
	}

	decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
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
	})

	// 2. RPC meta level messages.
	rpcMetaDecoratedEvents, err := m.parseRPCMeta(
		data.GetPeerId().String(),
		clientMeta,
		traceMeta,
		event,
		data.GetMeta(),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to parse rpc meta")
	}

	if len(rpcMetaDecoratedEvents) > 0 {
		decoratedEvents = append(decoratedEvents, rpcMetaDecoratedEvents...)
	}

	if err := m.handleNewDecoratedEvents(ctx, decoratedEvents); err != nil {
		return errors.Wrapf(err, "failed to handle decorated events")
	}

	return nil
}

func (m *Mimicry) handleDropRPCEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
	data, err := libp2p.TraceEventToDropRPC(event)
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

func (m *Mimicry) handlePublishMessageEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleRejectMessageEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleDuplicateMessageEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) handleDeliverMessageEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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

func (m *Mimicry) parseRPCMeta(
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	controlEvents, err := m.parseRPCMetaControl(
		peerID,
		clientMeta,
		traceMeta,
		event,
		data,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse rpc meta control")
	}

	decoratedEvents = append(decoratedEvents, controlEvents...)

	return decoratedEvents, nil
}

func (m *Mimicry) parseRPCMetaControl(
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	if data.GetControl().GetIhave() != nil {
		ihave, err := m.parseRPCMetaControlIHave(
			peerID,
			clientMeta,
			traceMeta,
			event,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control i have")
		}

		decoratedEvents = append(decoratedEvents, ihave...)
	}

	if data.GetControl().GetIwant() != nil {
		iwant, err := m.parseRPCMetaControlIWant(
			peerID,
			clientMeta,
			traceMeta,
			event,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control i want")
		}

		decoratedEvents = append(decoratedEvents, iwant...)
	}

	if data.GetControl().GetIdontwant() != nil {
		idontwant, err := m.parseRPCMetaControlIDontWant(
			peerID,
			clientMeta,
			traceMeta,
			event,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control idontwant")
		}

		decoratedEvents = append(decoratedEvents, idontwant...)
	}

	if data.GetControl().GetGraft() != nil {
		graft, err := m.parseRPCMetaControlGraft(
			peerID,
			clientMeta,
			traceMeta,
			event,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control graft")
		}

		decoratedEvents = append(decoratedEvents, graft...)
	}

	if data.GetControl().GetPrune() != nil {
		prune, err := m.parseRPCMetaControlPrune(
			peerID,
			clientMeta,
			traceMeta,
			event,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control prune")
		}

		decoratedEvents = append(decoratedEvents, prune...)
	}

	return decoratedEvents, nil
}

func (m *Mimicry) parseRPCMetaControlIHave(
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRpcMetaControlIhave{
		Libp2PTraceRpcMetaControlIhave: &xatu.ClientMeta_AdditionalLibP2PTraceRPCMetaControlIHaveData{
			Metadata: traceMeta,
		},
	}

	for _, ihave := range data.GetControl().GetIhave() {
		if !m.Config.Events.RpcMetaControlIHaveEnabled {
			continue
		}

		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE

		// Filter message IDs based on trace/sharding config.
		filteredMsgIDs, err := m.ShouldTraceRPCMetaMessages(
			event,
			clientMeta,
			eventType.String(),
			ihave.GetMessageIds(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for ihave: %w", err)
		}

		// Skip this control message if no message IDs remain after filtering.
		if len(filteredMsgIDs) == 0 {
			continue
		}

		for _, msgID := range filteredMsgIDs {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIhave{
					Libp2PTraceRpcMetaControlIhave: &libp2p.ControlIHaveMetaItem{
						PeerId:    wrapperspb.String(peerID),
						TopicId:   ihave.TopicId,
						MessageId: msgID,
					},
				},
			})
		}
	}

	return decoratedEvents, nil
}

func (m *Mimicry) parseRPCMetaControlIWant(
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRpcMetaControlIwant{
		Libp2PTraceRpcMetaControlIwant: &xatu.ClientMeta_AdditionalLibP2PTraceRPCMetaControlIWantData{
			Metadata: traceMeta,
		},
	}

	for _, iwant := range data.GetControl().GetIwant() {
		if !m.Config.Events.RpcMetaControlIWantEnabled {
			continue
		}

		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT

		// Filter message IDs based on trace/sharding config.
		filteredMsgIDs, err := m.ShouldTraceRPCMetaMessages(
			event,
			clientMeta,
			eventType.String(),
			iwant.GetMessageIds(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for iwant: %w", err)
		}

		// Skip this control message if no message IDs remain after filtering.
		if len(filteredMsgIDs) == 0 {
			continue
		}

		for _, msgID := range filteredMsgIDs {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIwant{
					Libp2PTraceRpcMetaControlIwant: &libp2p.ControlIWantMetaItem{
						PeerId:    wrapperspb.String(peerID),
						MessageId: msgID,
					},
				},
			})
		}
	}

	return decoratedEvents, nil
}

func (m *Mimicry) parseRPCMetaControlIDontWant(
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRpcMetaControlIdontwant{
		Libp2PTraceRpcMetaControlIdontwant: &xatu.ClientMeta_AdditionalLibP2PTraceRPCMetaControlIDontWantData{
			Metadata: traceMeta,
		},
	}

	for _, idontwant := range data.GetControl().GetIdontwant() {
		if !m.Config.Events.RpcMetaControlIDontWantEnabled {
			continue
		}

		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT

		// Filter message IDs based on trace/sharding config.
		filteredMsgIDs, err := m.ShouldTraceRPCMetaMessages(
			event,
			clientMeta,
			eventType.String(),
			idontwant.GetMessageIds(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for idontwant: %w", err)
		}

		// Skip this control message if no message IDs remain after filtering.
		if len(filteredMsgIDs) == 0 {
			continue
		}

		for _, msgID := range filteredMsgIDs {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIdontwant{
					Libp2PTraceRpcMetaControlIdontwant: &libp2p.ControlIDontWantMetaItem{
						PeerId:    wrapperspb.String(peerID),
						MessageId: msgID,
					},
				},
			})
		}
	}

	return decoratedEvents, nil
}

func (m *Mimicry) parseRPCMetaControlGraft(
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	decoratedEvents := make([]*xatu.DecoratedEvent, 0)

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRpcMetaControlGraft{
		Libp2PTraceRpcMetaControlGraft: &xatu.ClientMeta_AdditionalLibP2PTraceRPCMetaControlGraftData{
			Metadata: traceMeta,
		},
	}

	for _, graft := range data.GetControl().GetGraft() {
		if !m.Config.Events.RpcMetaControlGraftEnabled {
			continue
		}

		// How do we shard on this? We only have a topic ID.

		decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
			Event: &xatu.Event{
				Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT,
				DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
				Id:       uuid.New().String(),
			},
			Meta: &xatu.Meta{
				Client: metadata,
			},
			Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlGraft{
				Libp2PTraceRpcMetaControlGraft: &libp2p.ControlGraftMetaItem{
					PeerId:  wrapperspb.String(peerID),
					TopicId: graft.TopicId,
				},
			},
		})
	}

	return decoratedEvents, nil
}

func (m *Mimicry) parseRPCMetaControlPrune(
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, fmt.Errorf("failed to clone client metadata")
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRpcMetaControlPrune{
		Libp2PTraceRpcMetaControlPrune: &xatu.ClientMeta_AdditionalLibP2PTraceRPCMetaControlPruneData{
			Metadata: traceMeta,
		},
	}

	for _, prune := range data.GetControl().GetPrune() {
		if !m.Config.Events.RpcMetaControlPruneEnabled {
			continue
		}

		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE

		// Filter message IDs based on trace/sharding config.
		filteredPeerIDs, err := m.ShouldTraceRPCMetaMessages(
			event,
			clientMeta,
			eventType.String(),
			prune.GetPeerIds(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for prune: %w", err)
		}

		// Skip this control message if no message IDs remain after filtering.
		if len(filteredPeerIDs) == 0 {
			continue
		}

		for _, peerID := range filteredPeerIDs {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlPrune{
					Libp2PTraceRpcMetaControlPrune: &libp2p.ControlPruneMetaItem{
						PeerId:  peerID,
						TopicId: prune.TopicId,
					},
				},
			})
		}
	}

	return decoratedEvents, nil
}

// mapLibp2pEventToXatuEvent maps libp2p events to Xatu events.
func mapLibp2pEventToXatuEvent(event string) (string, error) {
	if xatuEvent, exists := libp2pToXatuEventMap[event]; exists {
		return xatuEvent, nil
	}

	return "", fmt.Errorf("unknown libp2p event: %s", event)
}
