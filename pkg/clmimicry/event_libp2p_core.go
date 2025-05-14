package clmimicry

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// handleHermesLibp2pCoreEvent handles libp2p core networking events.
// This includes CONNECTED and DISCONNECTED events from libp2p's network.Notify system.
func (m *Mimicry) handleHermesLibp2pCoreEvent(ctx context.Context, event *host.TraceEvent, clientMeta *xatu.ClientMeta, traceMeta *libp2p.TraceEventMetadata) error {
	// Extract MsgID for sampling decision
	msgID := getMsgID(event.Payload)

	switch event.Type {
	case "CONNECTED":
		if !m.Config.Events.Connected.Enabled {
			return nil
		}

		evtName := "CONNECTED"

		// Check if we should process this event based on sampling config
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		m.metrics.AddProcessedMessage(evtName)

		return m.handleConnectedEvent(ctx, clientMeta, traceMeta, event)

	case "DISCONNECTED":
		if !m.Config.Events.Disconnected.Enabled {
			return nil
		}

		evtName := "DISCONNECTED"

		// Check if we should process this event based on sampling config
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		m.metrics.AddProcessedMessage(evtName)

		return m.handleDisconnectedEvent(ctx, clientMeta, traceMeta, event)
	}

	return nil
}

func (m *Mimicry) handleConnectedEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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
	event *host.TraceEvent,
) error {
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
