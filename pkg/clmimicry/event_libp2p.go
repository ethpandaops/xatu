package clmimicry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (p *Processor) handleRemovePeerEvent(
	ctx context.Context,
	event *RemovePeerEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.RemovePeerEnabled {
		return nil
	}

	data := &libp2p.RemovePeer{
		PeerId: wrapperspb.String(event.GetPeerID().String()),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *JoinEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.JoinEnabled {
		return nil
	}

	data := &libp2p.Join{
		Topic: wrapperspb.String(event.Topic),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *LeaveEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.LeaveEnabled {
		return nil
	}

	data := &libp2p.Leave{
		Topic: wrapperspb.String(event.Topic),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *GraftEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.GraftEnabled {
		return nil
	}

	data := &libp2p.Graft{
		Topic:  wrapperspb.String(event.Topic),
		PeerId: wrapperspb.String(event.GetPeerID().String()),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *PruneEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.PruneEnabled {
		return nil
	}

	data := &libp2p.Prune{
		Topic:  wrapperspb.String(event.Topic),
		PeerId: wrapperspb.String(event.GetPeerID().String()),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *SendRPCEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	var (
		rootEventID     = uuid.New().String()
		decoratedEvents []*xatu.DecoratedEvent
		xatuEvent       = xatu.Event_LIBP2P_TRACE_SEND_RPC.String()
		networkStr      = getNetworkID(clientMeta)
	)

	peerID := event.GetPeerID().String()

	rpcMeta := convertRPCMetaToProto(event.Meta, peerID)

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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
				PeerId: wrapperspb.String(peerID),
				Meta: &libp2p.RPCMeta{
					PeerId: wrapperspb.String(peerID),
				},
			},
		},
	}

	// 2. RPC meta level messages.
	rpcMetaDecoratedEvents, err := p.parseRPCMetaFromTypedEvent(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		event.GetTimestamp(),
		rpcMeta,
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
	event *AddPeerEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.AddPeerEnabled {
		return nil
	}

	data := &libp2p.AddPeer{
		PeerId:   wrapperspb.String(event.GetPeerID().String()),
		Protocol: wrapperspb.String(string(event.Protocol)),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *RecvRPCEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	var (
		rootEventID     = uuid.New().String()
		decoratedEvents []*xatu.DecoratedEvent
		xatuEvent       = xatu.Event_LIBP2P_TRACE_RECV_RPC.String()
		networkStr      = getNetworkID(clientMeta)
	)

	peerID := event.GetPeerID().String()

	rpcMeta := convertRPCMetaToProto(event.Meta, peerID)

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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
				PeerId: wrapperspb.String(peerID),
				Meta: &libp2p.RPCMeta{
					PeerId: wrapperspb.String(peerID),
				},
			},
		},
	}

	// 2. RPC meta level messages.
	rpcMetaDecoratedEvents, err := p.parseRPCMetaFromTypedEvent(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		event.GetTimestamp(),
		rpcMeta,
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
	event *DropRPCEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	var (
		rootEventID     = uuid.New().String()
		decoratedEvents []*xatu.DecoratedEvent
		xatuEvent       = xatu.Event_LIBP2P_TRACE_DROP_RPC.String()
		networkStr      = getNetworkID(clientMeta)
	)

	peerID := event.GetPeerID().String()

	rpcMeta := convertRPCMetaToProto(event.Meta, peerID)

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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
				PeerId: wrapperspb.String(peerID),
				Meta: &libp2p.RPCMeta{
					PeerId: wrapperspb.String(peerID),
				},
			},
		},
	}

	// 2. RPC meta level messages.
	rpcMetaDecoratedEvents, err := p.parseRPCMetaFromTypedEvent(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		event.GetTimestamp(),
		rpcMeta,
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
	event *PublishMessageEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.PublishMessageEnabled {
		return nil
	}

	data := &libp2p.PublishMessage{
		MsgId: wrapperspb.String(event.MsgID),
		Topic: wrapperspb.String(event.Topic),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *RejectMessageEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.RejectMessageEnabled {
		return nil
	}

	data := &libp2p.RejectMessage{
		MsgId:     wrapperspb.String(event.MsgID),
		PeerId:    wrapperspb.String(event.GetPeerID().String()),
		Topic:     wrapperspb.String(event.Topic),
		Reason:    wrapperspb.String(event.Reason),
		Local:     wrapperspb.Bool(event.Local),
		MsgSize:   wrapperspb.UInt32(uint32(event.MsgSize)), //nolint:gosec // fine.
		SeqNumber: wrapperspb.UInt64(event.SeqNumber),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *DuplicateMessageEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.DuplicateMessageEnabled {
		return nil
	}

	data := &libp2p.DuplicateMessage{
		MsgId:     wrapperspb.String(event.MsgID),
		PeerId:    wrapperspb.String(event.GetPeerID().String()),
		Topic:     wrapperspb.String(event.Topic),
		Local:     wrapperspb.Bool(event.Local),
		MsgSize:   wrapperspb.UInt32(uint32(event.MsgSize)), //nolint:gosec // fine.
		SeqNumber: wrapperspb.UInt64(event.SeqNumber),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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
	event *DeliverMessageEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.DeliverMessageEnabled {
		return nil
	}

	data := &libp2p.DeliverMessage{
		MsgId:     wrapperspb.String(event.MsgID),
		PeerId:    wrapperspb.String(event.GetPeerID().String()),
		Topic:     wrapperspb.String(event.Topic),
		Local:     wrapperspb.Bool(event.Local),
		MsgSize:   wrapperspb.UInt32(uint32(event.MsgSize)), //nolint:gosec // fine.
		SeqNumber: wrapperspb.UInt64(event.SeqNumber),
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
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
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

// convertRPCMetaToProto converts RpcMeta to libp2p.RPCMeta protobuf format.
func convertRPCMetaToProto(meta *RpcMeta, peerID string) *libp2p.RPCMeta {
	if meta == nil {
		return &libp2p.RPCMeta{
			PeerId: wrapperspb.String(peerID),
		}
	}

	return &libp2p.RPCMeta{
		PeerId:        wrapperspb.String(peerID),
		Messages:      convertRPCMessages(meta.Messages),
		Subscriptions: convertRPCSubscriptions(meta.Subscriptions),
		Control:       convertRPCControl(meta.Control),
	}
}

// parseRPCMetaFromTypedEvent parses RPC meta from typed events.
func (p *Processor) parseRPCMetaFromTypedEvent(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	timestamp time.Time,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	controlEvents, err := p.parseRPCMetaControlFromTypedEvent(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		timestamp,
		data,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse rpc meta control")
	}

	subscriptionEvents, err := p.parseRPCMetaSubscriptionsFromTypedEvent(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		timestamp,
		data,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse rpc meta subscription")
	}

	messageEvents, err := p.parseRPCMetaMessagesFromTypedEvent(
		rootEventID,
		peerID,
		clientMeta,
		traceMeta,
		timestamp,
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

// parseRPCMetaControlFromTypedEvent parses RPC control metadata from typed events.
func (p *Processor) parseRPCMetaControlFromTypedEvent(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	timestamp time.Time,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	if data.GetControl().GetIhave() != nil && p.events.RpcMetaControlIHaveEnabled {
		ihave, err := p.parseRPCMetaControlIHaveFromTypedEvent(
			rootEventID,
			peerID,
			clientMeta,
			traceMeta,
			timestamp,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control i have")
		}

		decoratedEvents = append(decoratedEvents, ihave...)
	}

	if data.GetControl().GetIwant() != nil && p.events.RpcMetaControlIWantEnabled {
		iwant, err := p.parseRPCMetaControlIWantFromTypedEvent(
			context.Background(),
			timestamp,
			"LIBP2P_TRACE_RPC_META_CONTROL_IWANT",
			peerID,
			data,
			rootEventID,
			clientMeta,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control i want")
		}

		decoratedEvents = append(decoratedEvents, iwant...)
	}

	if data.GetControl().GetIdontwant() != nil && p.events.RpcMetaControlIDontWantEnabled {
		idontwant, err := p.parseRPCMetaControlIDontWantFromTypedEvent(
			context.Background(),
			timestamp,
			"LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT",
			peerID,
			data,
			rootEventID,
			clientMeta,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control idontwant")
		}

		decoratedEvents = append(decoratedEvents, idontwant...)
	}

	if data.GetControl().GetGraft() != nil && p.events.RpcMetaControlGraftEnabled {
		graft, err := p.parseRPCMetaControlGraftFromTypedEvent(
			context.Background(),
			timestamp,
			"LIBP2P_TRACE_RPC_META_CONTROL_GRAFT",
			peerID,
			data,
			rootEventID,
			clientMeta,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control graft")
		}

		decoratedEvents = append(decoratedEvents, graft...)
	}

	if data.GetControl().GetPrune() != nil && p.events.RpcMetaControlPruneEnabled {
		prune, err := p.parseRPCMetaControlPruneFromTypedEvent(
			context.Background(),
			timestamp,
			"LIBP2P_TRACE_RPC_META_CONTROL_PRUNE",
			peerID,
			data,
			rootEventID,
			clientMeta,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta control prune")
		}

		decoratedEvents = append(decoratedEvents, prune...)
	}

	return decoratedEvents, nil
}

// parseRPCMetaSubscriptionsFromTypedEvent parses RPC subscriptions from typed events.
func (p *Processor) parseRPCMetaSubscriptionsFromTypedEvent(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	timestamp time.Time,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	// Check if RPC meta subscriptions are enabled before processing
	if !p.events.RpcMetaSubscriptionEnabled {
		return decoratedEvents, nil
	}

	if data.GetSubscriptions() != nil {
		subscriptions, err := p.parseRPCMetaSubscriptionFromTypedEvent(
			context.Background(),
			timestamp,
			"LIBP2P_TRACE_RPC_META_SUBSCRIPTION",
			peerID,
			data,
			rootEventID,
			clientMeta,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta subscription")
		}

		decoratedEvents = append(decoratedEvents, subscriptions...)
	}

	return decoratedEvents, nil
}

// parseRPCMetaMessagesFromTypedEvent parses RPC messages from typed events.
func (p *Processor) parseRPCMetaMessagesFromTypedEvent(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	timestamp time.Time,
	data *libp2p.RPCMeta,
) ([]*xatu.DecoratedEvent, error) {
	var decoratedEvents []*xatu.DecoratedEvent

	// Check if RPC meta messages are enabled before processing
	if !p.events.RpcMetaMessageEnabled {
		return decoratedEvents, nil
	}

	if data.GetMessages() != nil {
		messages, err := p.parseRPCMetaMessageFromTypedEvent(
			context.Background(),
			timestamp,
			"LIBP2P_TRACE_RPC_META_MESSAGE",
			peerID,
			data,
			rootEventID,
			clientMeta,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rpc meta message")
		}

		decoratedEvents = append(decoratedEvents, messages...)
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaControlIHaveFromTypedEvent(
	rootEventID,
	peerID string,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	timestamp time.Time,
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
					DateTime: timestamppb.New(timestamp.Add(p.clockDrift)),
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

func (p *Processor) parseRPCMetaControlIWantFromTypedEvent(
	ctx context.Context,
	timestamp time.Time,
	eventType string,
	peerID string,
	rpcMeta *libp2p.RPCMeta,
	rootEventID string,
	metadata *xatu.ClientMeta,
) ([]*xatu.DecoratedEvent, error) {
	if rpcMeta.Control == nil {
		return nil, nil
	}

	decoratedEvents := make([]*xatu.DecoratedEvent, 0)

	for controlIndex, iwant := range rpcMeta.Control.Iwant {
		if iwant == nil {
			continue
		}

		messageInfos := make([]RPCMetaMessageInfo, 0, len(iwant.MessageIds))
		for _, msgID := range iwant.MessageIds {
			if msgID != nil {
				messageInfos = append(messageInfos, RPCMetaMessageInfo{
					MessageID: msgID,
				})
			}
		}

		filteredMessages, err := p.ShouldTraceRPCMetaMessages(metadata, eventType, messageInfos)
		if err != nil {
			return nil, fmt.Errorf("failed to filter IWANT messages: %w", err)
		}

		for _, msgWithIndex := range filteredMessages {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_Name(xatu.Event_Name_value[eventType]),
					DateTime: timestamppb.New(timestamp.Add(p.clockDrift)),
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

func (p *Processor) parseRPCMetaControlIDontWantFromTypedEvent(
	ctx context.Context,
	timestamp time.Time,
	eventType string,
	peerID string,
	rpcMeta *libp2p.RPCMeta,
	rootEventID string,
	metadata *xatu.ClientMeta,
) ([]*xatu.DecoratedEvent, error) {
	if rpcMeta.Control == nil {
		return nil, nil
	}

	decoratedEvents := make([]*xatu.DecoratedEvent, 0)

	for controlIndex, idontwant := range rpcMeta.Control.Idontwant {
		if idontwant == nil {
			continue
		}

		messageInfos := make([]RPCMetaMessageInfo, 0, len(idontwant.MessageIds))
		for _, msgID := range idontwant.MessageIds {
			if msgID != nil {
				messageInfos = append(messageInfos, RPCMetaMessageInfo{
					MessageID: msgID,
				})
			}
		}

		filteredMessages, err := p.ShouldTraceRPCMetaMessages(metadata, eventType, messageInfos)
		if err != nil {
			return nil, fmt.Errorf("failed to filter IDONTWANT messages: %w", err)
		}

		for _, msgWithIndex := range filteredMessages {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_Name(xatu.Event_Name_value[eventType]),
					DateTime: timestamppb.New(timestamp.Add(p.clockDrift)),
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

func (p *Processor) parseRPCMetaControlGraftFromTypedEvent(
	ctx context.Context,
	timestamp time.Time,
	eventType string,
	peerID string,
	rpcMeta *libp2p.RPCMeta,
	rootEventID string,
	metadata *xatu.ClientMeta,
) ([]*xatu.DecoratedEvent, error) {
	if rpcMeta.Control == nil {
		return nil, nil
	}

	decoratedEvents := make([]*xatu.DecoratedEvent, 0)

	for controlIndex, graft := range rpcMeta.Control.Graft {
		if graft == nil {
			continue
		}

		topicInfos := []RPCMetaTopicInfo{{Topic: graft.TopicId}}

		filteredTopics, err := p.ShouldTraceRPCMetaMessages(metadata, eventType, topicInfos)
		if err != nil {
			return nil, fmt.Errorf("failed to filter GRAFT topics: %w", err)
		}

		for range filteredTopics {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_Name(xatu.Event_Name_value[eventType]),
					DateTime: timestamppb.New(timestamp.Add(p.clockDrift)),
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
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaControlPruneFromTypedEvent(
	ctx context.Context,
	timestamp time.Time,
	eventType string,
	peerID string,
	rpcMeta *libp2p.RPCMeta,
	rootEventID string,
	metadata *xatu.ClientMeta,
) ([]*xatu.DecoratedEvent, error) {
	if rpcMeta.Control == nil {
		return nil, nil
	}

	decoratedEvents := make([]*xatu.DecoratedEvent, 0)

	for controlIndex, prune := range rpcMeta.Control.Prune {
		if prune == nil {
			continue
		}

		topicInfos := []RPCMetaTopicInfo{{Topic: prune.TopicId}}

		filteredTopics, err := p.ShouldTraceRPCMetaMessages(metadata, eventType, topicInfos)
		if err != nil {
			return nil, fmt.Errorf("failed to filter PRUNE topics: %w", err)
		}

		// Create one event per peer in the prune message
		// If no peer IDs are present, still emit one event for the topic
		if len(filteredTopics) > 0 {
			if len(prune.PeerIds) == 0 {
				// No peer IDs - emit one event for the topic
				decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
					Event: &xatu.Event{
						Name:     xatu.Event_Name(xatu.Event_Name_value[eventType]),
						DateTime: timestamppb.New(timestamp.Add(p.clockDrift)),
						Id:       uuid.New().String(),
					},
					Meta: &xatu.Meta{
						Client: metadata,
					},
					Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlPrune{
						Libp2PTraceRpcMetaControlPrune: &libp2p.ControlPruneMetaItem{
							RootEventId:  wrapperspb.String(rootEventID),
							PeerId:       wrapperspb.String(peerID),
							Topic:        prune.TopicId,
							ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
						},
					},
				})
			} else {
				// One event per peer ID
				for peerIndex, graftPeerID := range prune.PeerIds {
					decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
						Event: &xatu.Event{
							Name:     xatu.Event_Name(xatu.Event_Name_value[eventType]),
							DateTime: timestamppb.New(timestamp.Add(p.clockDrift)),
							Id:       uuid.New().String(),
						},
						Meta: &xatu.Meta{
							Client: metadata,
						},
						Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaControlPrune{
							Libp2PTraceRpcMetaControlPrune: &libp2p.ControlPruneMetaItem{
								RootEventId:  wrapperspb.String(rootEventID),
								PeerId:       wrapperspb.String(peerID),
								GraftPeerId:  graftPeerID,
								Topic:        prune.TopicId,
								ControlIndex: wrapperspb.UInt32(uint32(controlIndex)), //nolint:gosec // conversion fine.
								PeerIndex:    wrapperspb.UInt32(uint32(peerIndex)),    //nolint:gosec // conversion fine.
							},
						},
					})
				}
			}
		}
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaSubscriptionFromTypedEvent(
	ctx context.Context,
	timestamp time.Time,
	eventType string,
	peerID string,
	rpcMeta *libp2p.RPCMeta,
	rootEventID string,
	metadata *xatu.ClientMeta,
) ([]*xatu.DecoratedEvent, error) {
	decoratedEvents := make([]*xatu.DecoratedEvent, 0)

	for subIndex, sub := range rpcMeta.Subscriptions {
		if sub == nil {
			continue
		}

		topicInfos := []RPCMetaTopicInfo{{Topic: sub.TopicId}}

		filteredTopics, err := p.ShouldTraceRPCMetaMessages(metadata, eventType, topicInfos)
		if err != nil {
			return nil, fmt.Errorf("failed to filter subscription topics: %w", err)
		}

		for range filteredTopics {
			decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_Name(xatu.Event_Name_value[eventType]),
					DateTime: timestamppb.New(timestamp.Add(p.clockDrift)),
					Id:       uuid.New().String(),
				},
				Meta: &xatu.Meta{
					Client: metadata,
				},
				Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaSubscription{
					Libp2PTraceRpcMetaSubscription: &libp2p.SubMetaItem{
						RootEventId:  wrapperspb.String(rootEventID),
						PeerId:       wrapperspb.String(peerID),
						TopicId:      sub.TopicId,
						Subscribe:    sub.Subscribe,
						ControlIndex: wrapperspb.UInt32(uint32(subIndex)), //nolint:gosec // conversion fine.
					},
				},
			})
		}
	}

	return decoratedEvents, nil
}

func (p *Processor) parseRPCMetaMessageFromTypedEvent(
	ctx context.Context,
	timestamp time.Time,
	eventType string,
	peerID string,
	rpcMeta *libp2p.RPCMeta,
	rootEventID string,
	metadata *xatu.ClientMeta,
) ([]*xatu.DecoratedEvent, error) {
	decoratedEvents := make([]*xatu.DecoratedEvent, 0)

	messageInfos := make([]RPCMetaMessageInfo, 0, len(rpcMeta.Messages))
	for _, msg := range rpcMeta.Messages {
		if msg != nil {
			messageInfos = append(messageInfos, RPCMetaMessageInfo{
				MessageID: msg.MessageId,
				Topic:     msg.TopicId,
			})
		}
	}

	filteredMessages, err := p.ShouldTraceRPCMetaMessages(metadata, eventType, messageInfos)
	if err != nil {
		return nil, fmt.Errorf("failed to filter RPC messages: %w", err)
	}

	for _, msgWithIndex := range filteredMessages {
		decoratedEvents = append(decoratedEvents, &xatu.DecoratedEvent{
			Event: &xatu.Event{
				Name:     xatu.Event_Name(xatu.Event_Name_value[eventType]),
				DateTime: timestamppb.New(timestamp.Add(p.clockDrift)),
				Id:       uuid.New().String(),
			},
			Meta: &xatu.Meta{
				Client: metadata,
			},
			Data: &xatu.DecoratedEvent_Libp2PTraceRpcMetaMessage{
				Libp2PTraceRpcMetaMessage: &libp2p.MessageMetaItem{
					RootEventId:  wrapperspb.String(rootEventID),
					PeerId:       wrapperspb.String(peerID),
					MessageId:    msgWithIndex.MessageID,
					TopicId:      rpcMeta.Messages[msgWithIndex.OriginalIndex].TopicId,
					ControlIndex: wrapperspb.UInt32(msgWithIndex.OriginalIndex),
				},
			},
		})
	}

	return decoratedEvents, nil
}

func convertRPCMessages(messages []RpcMetaMsg) []*libp2p.MessageMeta {
	ourMessages := make([]*libp2p.MessageMeta, len(messages))

	for i, msg := range messages {
		ourMessages[i] = &libp2p.MessageMeta{
			MessageId: wrapperspb.String(msg.MsgID),
			TopicId:   wrapperspb.String(msg.Topic),
		}
	}

	return ourMessages
}

func convertRPCSubscriptions(subs []RpcMetaSub) []*libp2p.SubMeta {
	ourSubs := make([]*libp2p.SubMeta, len(subs))

	for i, sub := range subs {
		ourSubs[i] = &libp2p.SubMeta{
			Subscribe: wrapperspb.Bool(sub.Subscribe),
			TopicId:   wrapperspb.String(sub.TopicID),
		}
	}

	return ourSubs
}

func convertRPCControl(ctrl *RpcMetaControl) *libp2p.ControlMeta {
	if ctrl == nil {
		return nil
	}

	return &libp2p.ControlMeta{
		Ihave:     convertControlIHaveMeta(ctrl.IHave),
		Iwant:     convertControlIWantMeta(ctrl.IWant),
		Graft:     convertControlGraftMeta(ctrl.Graft),
		Prune:     convertControlPruneMeta(ctrl.Prune),
		Idontwant: convertControlIDontWantMeta(ctrl.Idontwant),
	}
}

func convertControlIHaveMeta(ihave []RpcControlIHave) []*libp2p.ControlIHaveMeta {
	converted := make([]*libp2p.ControlIHaveMeta, len(ihave))

	for i, item := range ihave {
		converted[i] = &libp2p.ControlIHaveMeta{
			TopicId:    wrapperspb.String(item.TopicID),
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlIWantMeta(iwant []RpcControlIWant) []*libp2p.ControlIWantMeta {
	converted := make([]*libp2p.ControlIWantMeta, len(iwant))

	for i, item := range iwant {
		converted[i] = &libp2p.ControlIWantMeta{
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlIDontWantMeta(idontwant []RpcControlIdontWant) []*libp2p.ControlIDontWantMeta {
	converted := make([]*libp2p.ControlIDontWantMeta, len(idontwant))

	for i, item := range idontwant {
		converted[i] = &libp2p.ControlIDontWantMeta{
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlGraftMeta(graft []RpcControlGraft) []*libp2p.ControlGraftMeta {
	converted := make([]*libp2p.ControlGraftMeta, len(graft))

	for i, item := range graft {
		converted[i] = &libp2p.ControlGraftMeta{
			TopicId: wrapperspb.String(item.TopicID),
		}
	}

	return converted
}

func convertControlPruneMeta(prune []RpcControlPrune) []*libp2p.ControlPruneMeta {
	converted := make([]*libp2p.ControlPruneMeta, len(prune))

	for i, item := range prune {
		peerIds := make([]string, 0, len(item.PeerIDs))
		for _, peer := range item.PeerIDs {
			peerIds = append(peerIds, peer.String())
		}

		converted[i] = &libp2p.ControlPruneMeta{
			TopicId: wrapperspb.String(item.TopicID),
			PeerIds: convertStringValues(peerIds),
		}
	}

	return converted
}

func convertStringValues(strings []string) []*wrapperspb.StringValue {
	converted := make([]*wrapperspb.StringValue, len(strings))

	for i, s := range strings {
		converted[i] = wrapperspb.String(s)
	}

	return converted
}
