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

// handleHermesRPCEvent handles Request/Response (RPC) protocol events.
func (m *Mimicry) handleHermesRPCEvent(
	ctx context.Context,
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	// Extract MsgID for sampling decision
	msgID := getMsgID(event.Payload)

	// Extract network from clientMeta
	network := clientMeta.GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = unknown
	}

	switch event.Type {
	case TraceEvent_HANDLE_METADATA:
		if !m.Config.Events.HandleMetadataEnabled {
			return nil
		}

		// Check if we should process this event based on sampling config
		if msgID != "" && !m.ShouldTraceMessage(msgID, TraceEvent_HANDLE_METADATA, networkStr) {
			m.metrics.AddSkippedMessage(TraceEvent_HANDLE_METADATA, networkStr)

			return nil
		}

		m.metrics.AddProcessedMessage(TraceEvent_HANDLE_METADATA, networkStr)

		return m.handleHandleMetadataEvent(ctx, clientMeta, traceMeta, event)

	case TraceEvent_HANDLE_STATUS:
		if !m.Config.Events.HandleStatusEnabled {
			return nil
		}

		// Check if we should process this event based on sampling config
		if msgID != "" && !m.ShouldTraceMessage(msgID, TraceEvent_HANDLE_STATUS, networkStr) {
			m.metrics.AddSkippedMessage(TraceEvent_HANDLE_STATUS, networkStr)

			return nil
		}

		m.metrics.AddProcessedMessage(TraceEvent_HANDLE_STATUS, networkStr)

		return m.handleHandleStatusEvent(ctx, clientMeta, traceMeta, event)
	}

	return nil
}

func (m *Mimicry) handleHandleMetadataEvent(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
	event *host.TraceEvent,
) error {
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
	event *host.TraceEvent,
) error {
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
