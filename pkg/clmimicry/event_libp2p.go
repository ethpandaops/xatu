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

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// handleHermesLibp2pEvent handles libp2p pubsub protocol level events.
func (m *Mimicry) handleHermesLibp2pEvent(
	ctx context.Context,
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	// Extract MsgID for sampling decision.
	msgID := getMsgID(event.Payload)

	switch event.Type {
	case pubsubpb.TraceEvent_ADD_PEER.String():
		if !m.Config.Events.AddPeer.Enabled {
			return nil
		}

		evtName := pubsubpb.TraceEvent_ADD_PEER.String()

		// Check if we should process this event based on sampling config.
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		m.metrics.AddProcessedMessage(evtName)

		return m.handleAddPeerEvent(ctx, clientMeta, traceMeta, event)

	case pubsubpb.TraceEvent_RECV_RPC.String():
		if !m.Config.Events.RecvRPC.Enabled {
			return nil
		}

		evtName := pubsubpb.TraceEvent_RECV_RPC.String()

		// Check if we should process this event based on sampling config.
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		m.metrics.AddProcessedMessage(evtName)

		return m.handleRecvRPCEvent(ctx, clientMeta, traceMeta, event)

	case pubsubpb.TraceEvent_SEND_RPC.String():
		if !m.Config.Events.SendRPC.Enabled {
			return nil
		}

		evtName := pubsubpb.TraceEvent_SEND_RPC.String()

		// Check if we should process this event based on sampling config.
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		m.metrics.AddProcessedMessage(evtName)

		return m.handleSendRPCEvent(ctx, clientMeta, traceMeta, event)

	case pubsubpb.TraceEvent_REMOVE_PEER.String():
		if !m.Config.Events.RemovePeer.Enabled {
			return nil
		}

		evtName := pubsubpb.TraceEvent_REMOVE_PEER.String()

		// Check if we should process this event based on sampling config.
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		m.metrics.AddProcessedMessage(evtName)

		return m.handleRemovePeerEvent(ctx, clientMeta, traceMeta, event)

	case pubsubpb.TraceEvent_JOIN.String():
		if !m.Config.Events.Join.Enabled {
			return nil
		}

		evtName := pubsubpb.TraceEvent_JOIN.String()

		// Check if we should process this event based on sampling config.
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		m.metrics.AddProcessedMessage(evtName)

		return m.handleJoinEvent(ctx, clientMeta, traceMeta, event)
	}

	return nil
}

func (m *Mimicry) handleRemovePeerEvent(ctx context.Context,
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

func (m *Mimicry) handleJoinEvent(ctx context.Context,
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

func (m *Mimicry) handleSendRPCEvent(ctx context.Context,
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

func (m *Mimicry) handleAddPeerEvent(ctx context.Context,
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

func (m *Mimicry) handleRecvRPCEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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
