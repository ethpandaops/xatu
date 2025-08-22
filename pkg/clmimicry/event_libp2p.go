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
//
//nolint:gocyclo // This function handles multiple event types and is intentionally complex
func (p *Processor) handleHermesLibp2pEvent(
	ctx context.Context,
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	// Map libp2p event to Xatu event.
	xatuEvent, err := mapLibp2pEventToXatuEvent(event.Type)
	if err != nil {
		p.log.WithField("event", event.Type).Tracef("unsupported event in handleHermesLibp2pEvent event")

		//nolint:nilerr // we don't want to return an error here.
		return nil
	}

	networkStr := getNetworkID(clientMeta)

	switch xatuEvent {
	case xatu.Event_LIBP2P_TRACE_ADD_PEER.String():
		if !p.events.AddPeerEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleAddPeerEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_RECV_RPC.String():
		// Always process RPC events to extract child events, even if parent is disabled
		// This allows child events (like IHAVE) to be captured independently
		return p.handleRecvRPCEvent(ctx, clientMeta, traceMeta, event, xatuEvent, networkStr)

	case xatu.Event_LIBP2P_TRACE_DROP_RPC.String():
		// Always process RPC events to extract child events, even if parent is disabled
		// This allows child events (like IHAVE) to be captured independently
		return p.handleDropRPCEvent(ctx, clientMeta, traceMeta, event, xatuEvent, networkStr)

	case xatu.Event_LIBP2P_TRACE_SEND_RPC.String():
		// Always process RPC events to extract child events, even if parent is disabled
		// This allows child events (like IHAVE) to be captured independently
		return p.handleSendRPCEvent(ctx, clientMeta, traceMeta, event, xatuEvent, networkStr)

	case xatu.Event_LIBP2P_TRACE_REMOVE_PEER.String():
		if !p.events.RemovePeerEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleRemovePeerEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_JOIN.String():
		if !p.events.JoinEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleJoinEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_LEAVE.String():
		if !p.events.LeaveEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleLeaveEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_GRAFT.String():
		if !p.events.GraftEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleGraftEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_PRUNE.String():
		if !p.events.PruneEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handlePruneEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE.String():
		if !p.events.PublishMessageEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handlePublishMessageEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE.String():
		if !p.events.RejectMessageEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleRejectMessageEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String():
		if !p.events.DuplicateMessageEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleDuplicateMessageEvent(ctx, clientMeta, traceMeta, event)
	case xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE.String():
		if !p.events.DeliverMessageEnabled {
			return nil
		}

		// Record that we received this event
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this event based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		return p.handleDeliverMessageEvent(ctx, clientMeta, traceMeta, event)
	}

	return nil
}

func (p *Processor) handleRemovePeerEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceRemovePeer{
			Libp2PTraceRemovePeer: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleJoinEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceJoin{
			Libp2PTraceJoin: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleLeaveEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceLeave{
			Libp2PTraceLeave: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleGraftEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGraft{
			Libp2PTraceGraft: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handlePruneEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTracePrune{
			Libp2PTracePrune: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleSendRPCEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	xatuEvent string,
	networkStr string,
) error {
	var (
		rootEventID     = uuid.New().String()
		decoratedEvents []*xatu.DecoratedEvent
	)

	data, err := libp2p.TraceEventToSendRPC(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to deliver message event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	// 1. Root level rpc event.
	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceSendRpc{
		Libp2PTraceSendRpc: &xatu.ClientMeta_AdditionalLibP2PTraceSendRPCData{
			Metadata: traceMeta,
		},
	}

	rootRPCEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_SEND_RPC,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       rootEventID,
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		// The root level event will still contain all rpc meta level messages. With some events, these
		// are really large, and will exceed msg limits at kafka/vector. Because we're splitting the meta
		// events into multiple messages, we can remove the meta level messages from the root event.
		Data: &xatu.DecoratedEvent_Libp2PTraceSendRpc{
			Libp2PTraceSendRpc: &libp2p.SendRPC{
				PeerId: data.GetPeerId(),
				Meta: &libp2p.RPCMeta{
					PeerId: data.GetPeerId(),
				},
			},
		},
	}

	// 2. RPC meta level messages.
	rpcMetaDecoratedEvents, err := p.parseRPCMeta(
		rootEventID,
		data.GetPeerId().GetValue(),
		clientMeta,
		traceMeta,
		event,
		data.GetMeta(),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to parse rpc meta")
	}

	// Only send events if there are child events to send
	// The parent event is only sent alongside child events, not standalone
	if len(rpcMetaDecoratedEvents) > 0 {
		// Check if parent event should also be included
		if p.events.SendRPCEnabled {
			p.metrics.AddEvent(xatuEvent, networkStr)

			if p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
				decoratedEvents = append(decoratedEvents, rootRPCEvent)
			}
		}

		decoratedEvents = append(decoratedEvents, rpcMetaDecoratedEvents...)

		if err := p.output.HandleDecoratedEvents(ctx, decoratedEvents); err != nil {
			return errors.Wrapf(err, "failed to handle decorated events")
		}
	}

	return nil
}

func (p *Processor) handleAddPeerEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceAddPeer{
			Libp2PTraceAddPeer: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleRecvRPCEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	xatuEvent string,
	networkStr string,
) error {
	var (
		rootEventID     = uuid.New().String()
		decoratedEvents []*xatu.DecoratedEvent
	)

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

	rootRPCEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_RECV_RPC,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       rootEventID,
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		// The root level event will still contain all rpc meta level messages. With some events, these
		// are really large, and will exceed msg limits at kafka/vector. Because we're splitting the meta
		// events into multiple messages, we can remove the meta level messages from the root event.
		Data: &xatu.DecoratedEvent_Libp2PTraceRecvRpc{
			Libp2PTraceRecvRpc: &libp2p.RecvRPC{
				PeerId: data.GetPeerId(),
				Meta: &libp2p.RPCMeta{
					PeerId: data.GetPeerId(),
				},
			},
		},
	}

	// 2. RPC meta level messages.
	rpcMetaDecoratedEvents, err := p.parseRPCMeta(
		rootEventID,
		data.GetPeerId().GetValue(),
		clientMeta,
		traceMeta,
		event,
		data.GetMeta(),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to parse rpc meta")
	}

	// Only send events if there are child events to send
	// The parent event is only sent alongside child events, not standalone
	if len(rpcMetaDecoratedEvents) > 0 {
		// Check if parent event should also be included
		if p.events.RecvRPCEnabled {
			p.metrics.AddEvent(xatuEvent, networkStr)

			if p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
				decoratedEvents = append(decoratedEvents, rootRPCEvent)
			}
		}

		decoratedEvents = append(decoratedEvents, rpcMetaDecoratedEvents...)

		if err := p.output.HandleDecoratedEvents(ctx, decoratedEvents); err != nil {
			return errors.Wrapf(err, "failed to handle decorated events")
		}
	}

	return nil
}

func (p *Processor) handleDropRPCEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	xatuEvent string,
	networkStr string,
) error {
	var (
		rootEventID     = uuid.New().String()
		decoratedEvents []*xatu.DecoratedEvent
	)

	data, err := libp2p.TraceEventToDropRPC(event)
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

	rootRPCEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_DROP_RPC,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       rootEventID,
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		// The root level event will still contain all rpc meta level messages. With some events, these
		// are really large, and will exceed msg limits at kafka/vector. Because we're splitting the meta
		// events into multiple messages, we can remove the meta level messages from the root event.
		Data: &xatu.DecoratedEvent_Libp2PTraceDropRpc{
			Libp2PTraceDropRpc: &libp2p.DropRPC{
				PeerId: data.GetPeerId(),
				Meta: &libp2p.RPCMeta{
					PeerId: data.GetPeerId(),
				},
			},
		},
	}

	// 2. RPC meta level messages.
	rpcMetaDecoratedEvents, err := p.parseRPCMeta(
		rootEventID,
		data.GetPeerId().GetValue(),
		clientMeta,
		traceMeta,
		event,
		data.GetMeta(),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to parse rpc meta")
	}

	// Only send events if there are child events to send
	// The parent event is only sent alongside child events, not standalone
	if len(rpcMetaDecoratedEvents) > 0 {
		// Check if parent event should also be included
		if p.events.DropRPCEnabled {
			p.metrics.AddEvent(xatuEvent, networkStr)

			if p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
				decoratedEvents = append(decoratedEvents, rootRPCEvent)
			}
		}

		decoratedEvents = append(decoratedEvents, rpcMetaDecoratedEvents...)

		if err := p.output.HandleDecoratedEvents(ctx, decoratedEvents); err != nil {
			return errors.Wrapf(err, "failed to handle decorated events")
		}
	}

	return nil
}

func (p *Processor) handlePublishMessageEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTracePublishMessage{
			Libp2PTracePublishMessage: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleRejectMessageEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceRejectMessage{
			Libp2PTraceRejectMessage: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleDuplicateMessageEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceDuplicateMessage{
			Libp2PTraceDuplicateMessage: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) handleDeliverMessageEvent(
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
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceDeliverMessage{
			Libp2PTraceDeliverMessage: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) parseRPCMeta(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	controlEvents, err := p.parseRPCMetaControl(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		event,
		data,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse rpc meta control")
	}

	subscriptionEvents, err := p.parseRPCMetaSubscriptions(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		event,
		data,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse rpc meta subscription")
	}

	messageEvents, err := p.parseRPCMetaMessages(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		event,
		data,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse rpc meta message")
	}

	decoratedEvents = append(decoratedEvents, controlEvents...)
	decoratedEvents = append(decoratedEvents, subscriptionEvents...)
	decoratedEvents = append(decoratedEvents, messageEvents...)

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaControl(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	if data.GetControl().GetIhave() != nil && p.events.RpcMetaControlIHaveEnabled {
		ihave, err := p.parseRPCMetaControlIHave(
			rootEventID,
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

	if data.GetControl().GetIwant() != nil && p.events.RpcMetaControlIWantEnabled {
		iwant, err := p.parseRPCMetaControlIWant(
			rootEventID,
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

	if data.GetControl().GetIdontwant() != nil && p.events.RpcMetaControlIDontWantEnabled {
		idontwant, err := p.parseRPCMetaControlIDontWant(
			rootEventID,
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

	if data.GetControl().GetGraft() != nil && p.events.RpcMetaControlGraftEnabled {
		graft, err := p.parseRPCMetaControlGraft(
			rootEventID,
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

	if data.GetControl().GetPrune() != nil && p.events.RpcMetaControlPruneEnabled {
		prune, err := p.parseRPCMetaControlPrune(
			rootEventID,
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

func (p *Processor) parseRPCMetaControlIHave(
	rootEventID,
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

	for controlIndex, ihave := range data.GetControl().GetIhave() {
		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE

		// Create topic-aware message info for hierarchical sharding support
		messageInfos := make([]RPCMetaMessageInfo, len(ihave.GetMessageIds()))
		for i, msgID := range ihave.GetMessageIds() {
			messageInfos[i] = RPCMetaMessageInfo{
				MessageID: msgID,
				Topic:     ihave.GetTopicId(), // Use the topic_id from IHAVE for hierarchical sharding
			}
		}

		// Filter message IDs based on trace/sharding config, preserving original indices.
		filteredMsgIDsWithIndex, err := p.ShouldTraceRPCMetaMessages(
			clientMeta,
			eventType.String(),
			messageInfos,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for ihave: %w", err)
		}

		// Skip this control message if no message IDs remain after filtering.
		if len(filteredMsgIDsWithIndex) == 0 {
			continue
		}

		for _, msgWithIndex := range filteredMsgIDsWithIndex {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIhave{
					Libp2PTraceRpcMetaControlIhave: &libp2p.ControlIHaveMetaItem{
						RootEventId:  wrapperspb.String(rootEventID),
						PeerId:       wrapperspb.String(peerID),
						Topic:        ihave.TopicId,
						MessageId:    msgWithIndex.MessageID,
						MessageIndex: wrapperspb.UInt32(msgWithIndex.OriginalIndex),
						ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
					},
				},
			})
		}
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaControlIWant(
	rootEventID,
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

	for controlIndex, iwant := range data.GetControl().GetIwant() {
		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT

		// Filter message IDs based on trace/sharding config, preserving original indices.
		filteredMsgIDsWithIndex, err := p.ShouldTraceRPCMetaMessages(
			clientMeta,
			eventType.String(),
			iwant.GetMessageIds(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for iwant: %w", err)
		}

		// Skip this control message if no message IDs remain after filtering.
		if len(filteredMsgIDsWithIndex) == 0 {
			continue
		}

		for _, msgWithIndex := range filteredMsgIDsWithIndex {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIwant{
					Libp2PTraceRpcMetaControlIwant: &libp2p.ControlIWantMetaItem{
						RootEventId:  wrapperspb.String(rootEventID),
						PeerId:       wrapperspb.String(peerID),
						MessageId:    msgWithIndex.MessageID,
						MessageIndex: wrapperspb.UInt32(msgWithIndex.OriginalIndex),
						ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
					},
				},
			})
		}
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaControlIDontWant(
	rootEventID,
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

	for controlIndex, idontwant := range data.GetControl().GetIdontwant() {
		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT

		// Filter message IDs based on trace/sharding config, preserving original indices.
		filteredMsgIDsWithIndex, err := p.ShouldTraceRPCMetaMessages(
			clientMeta,
			eventType.String(),
			idontwant.GetMessageIds(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for idontwant: %w", err)
		}

		// Skip this control message if no message IDs remain after filtering.
		if len(filteredMsgIDsWithIndex) == 0 {
			continue
		}

		for _, msgWithIndex := range filteredMsgIDsWithIndex {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlIdontwant{
					Libp2PTraceRpcMetaControlIdontwant: &libp2p.ControlIDontWantMetaItem{
						RootEventId:  wrapperspb.String(rootEventID),
						PeerId:       wrapperspb.String(peerID),
						MessageId:    msgWithIndex.MessageID,
						MessageIndex: wrapperspb.UInt32(msgWithIndex.OriginalIndex),
						ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
					},
				},
			})
		}
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaControlGraft(
	rootEventID,
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

	for controlIndex, graft := range data.GetControl().GetGraft() {
		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT

		// Create topic info for Group B event sharding (topic-based)
		topicInfo := RPCMetaTopicInfo{
			Topic: graft.TopicId,
		}

		// Filter based on trace/sharding config using the same method as other RPC meta messages.
		// GRAFT is a Group B event (sharded by topic only).
		filteredTopicsWithIndex, err := p.ShouldTraceRPCMetaMessages(
			clientMeta,
			eventType.String(),
			[]RPCMetaTopicInfo{topicInfo},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for graft: %w", err)
		}

		// Skip this control message if filtered out.
		if len(filteredTopicsWithIndex) == 0 {
			continue
		}

		decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
			Event: &xatu.Event{
				Name:     xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT,
				DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
				Id:       uuid.New().String(),
			},
			Meta: &xatu.Meta{
				Client: metadata,
			},
			Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlGraft{
				Libp2PTraceRpcMetaControlGraft: &libp2p.ControlGraftMetaItem{
					RootEventId:  wrapperspb.String(rootEventID),
					PeerId:       wrapperspb.String(peerID),
					Topic:        graft.TopicId,
					ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
				},
			},
		})
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaControlPrune(
	rootEventID,
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

	for controlIndex, prune := range data.GetControl().GetPrune() {
		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE

		// Create topic info for Group B event sharding (topic-based)
		topicInfo := RPCMetaTopicInfo{
			Topic: prune.TopicId,
		}

		// Filter based on trace/sharding config using the same method as other RPC meta messages.
		// PRUNE is a Group B event (sharded by topic only).
		filteredTopicsWithIndex, err := p.ShouldTraceRPCMetaMessages(
			clientMeta,
			eventType.String(),
			[]RPCMetaTopicInfo{topicInfo},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for prune: %w", err)
		}

		// Skip this control message if filtered out.
		if len(filteredTopicsWithIndex) == 0 {
			continue
		}

		// PRUNE messages may or may not contain peer IDs depending on whether mesh participants
		// have opted into GossipSub PX (Peer eXchange) and are running v1.1 or higher.
		//
		// When peer IDs are present:
		//   - We create one event per peer ID to maintain granular tracking
		//   - Each event includes the specific peer ID.
		//
		// When peer IDs are absent (most common case):
		//   - We still need to record the PRUNE event for network analysis
		//   - We create a single event with an empty GraftPeerId
		//   - This ensures we capture all PRUNE activity even without PX data

		peerIds := prune.GetPeerIds()
		if len(peerIds) == 0 {
			// No peer IDs present - create a single event with empty GraftPeerId
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlPrune{
					Libp2PTraceRpcMetaControlPrune: &libp2p.ControlPruneMetaItem{
						RootEventId:  wrapperspb.String(rootEventID),
						GraftPeerId:  nil, // Explicitly set to nil when no peer IDs are provided
						PeerId:       wrapperspb.String(peerID),
						Topic:        prune.TopicId,
						PeerIndex:    wrapperspb.UInt32(0),                    // Always 0 when no peer IDs
						ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
					},
				},
			})
		} else {
			// Peer IDs present - create one event per peer ID
			for peerIndex, prunePeerID := range peerIds {
				if prunePeerID == nil || prunePeerID.GetValue() == "" {
					continue
				}

				decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
					Event: &xatu.Event{
						Name:     eventType,
						DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
						Id:       uuid.New().String(),
					},
					Meta: &xatu.Meta{
						Client: metadata,
					},
					Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlPrune{
						Libp2PTraceRpcMetaControlPrune: &libp2p.ControlPruneMetaItem{
							RootEventId:  wrapperspb.String(rootEventID),
							GraftPeerId:  prunePeerID,
							PeerId:       wrapperspb.String(peerID),
							Topic:        prune.TopicId,
							PeerIndex:    wrapperspb.UInt32(uint32(peerIndex)),    //nolint:gosec // conversion fine.
							ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
						},
					},
				})
			}
		}
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaSubscriptions(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	// Check if RPC meta subscriptions are enabled before processing
	if !p.events.RpcMetaSubscriptionEnabled {
		return decoratedEvents, nil
	}

	if data.GetSubscriptions() != nil {
		subscriptions, err := p.parseRPCMetaSubscription(
			rootEventID,
			peerID,
			clientMeta,
			traceMeta,
			event,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta subscription")
		}

		decoratedEvents = append(decoratedEvents, subscriptions...)
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaSubscription(
	rootEventID,
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

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRpcMetaSubscription{
		Libp2PTraceRpcMetaSubscription: &xatu.ClientMeta_AdditionalLibP2PTraceRPCMetaSubscriptionData{
			Metadata: traceMeta,
		},
	}

	for controlIndex, subscription := range data.GetSubscriptions() {
		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION

		// Create topic info for Group B event sharding (topic-based)
		topicInfo := RPCMetaTopicInfo{
			Topic: subscription.TopicId,
		}

		// Filter based on trace/sharding config using the same method as other RPC meta messages.
		// SUBSCRIPTION is a Group B event (sharded by topic only).
		filteredTopicsWithIndex, err := p.ShouldTraceRPCMetaMessages(
			clientMeta,
			eventType.String(),
			[]RPCMetaTopicInfo{topicInfo},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for subscription: %w", err)
		}

		// Skip this control message if filtered out.
		if len(filteredTopicsWithIndex) == 0 {
			continue
		}

		decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
			Event: &xatu.Event{
				Name:     eventType,
				DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
				Id:       uuid.New().String(),
			},
			Meta: &xatu.Meta{
				Client: metadata,
			},
			Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaSubscription{
				Libp2PTraceRpcMetaSubscription: &libp2p.SubMetaItem{
					RootEventId:  wrapperspb.String(rootEventID),
					PeerId:       wrapperspb.String(peerID),
					TopicId:      subscription.TopicId,
					Subscribe:    subscription.Subscribe,
					ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
				},
			},
		})
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaMessages(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	// Check if RPC meta messages are enabled before processing
	if !p.events.RpcMetaMessageEnabled {
		return decoratedEvents, nil
	}

	if data.GetMessages() != nil {
		messages, err := p.parseRPCMetaMessage(
			rootEventID,
			peerID,
			clientMeta,
			traceMeta,
			event,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta message")
		}

		decoratedEvents = append(decoratedEvents, messages...)
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaMessage(
	rootEventID,
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

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceRpcMetaMessage{
		Libp2PTraceRpcMetaMessage: &xatu.ClientMeta_AdditionalLibP2PTraceRPCMetaMessageData{
			Metadata: traceMeta,
		},
	}

	for controlIndex, message := range data.GetMessages() {
		eventType := xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE

		// Filter messages with topic-aware hierarchical sharding support.
		filteredMsgIDsWithIndex, err := p.ShouldTraceRPCMetaMessages(
			clientMeta,
			eventType.String(),
			[]RPCMetaMessageInfo{
				{
					MessageID: message.MessageId,
					Topic:     message.TopicId, // Include topic information for hierarchical filtering
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check trace config for message: %w", err)
		}

		// Skip this control message if no message IDs remain after filtering.
		if len(filteredMsgIDsWithIndex) == 0 {
			continue
		}

		for range filteredMsgIDsWithIndex {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventType,
					DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaMessage{
					Libp2PTraceRpcMetaMessage: &libp2p.MessageMetaItem{
						RootEventId:  wrapperspb.String(rootEventID),
						PeerId:       wrapperspb.String(peerID),
						MessageId:    message.MessageId,
						TopicId:      message.TopicId,
						ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
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
