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
	// Extract MsgID for sharding decision.
	msgID := getMsgID(event.Payload)

	// Map libp2p event to Xatu event.
	xatuEvent, err := mapRPCEventToXatuEvent(event.Type)
	if err != nil {
		return errors.Wrapf(err, "failed to map RPC event to Xatu event")
	}

	// Extract network from clientMeta
	network := clientMeta.GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = unknown
	}

	switch xatuEvent {
	case xatu.Event_LIBP2P_TRACE_HANDLE_METADATA.String():
		if !m.Config.Events.HandleMetadataEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config.
		if msgID != "" && !m.ShouldTraceMessage(msgID, xatuEvent, networkStr) {
			m.metrics.AddSkippedMessage(xatuEvent, networkStr)

			return nil
		}

		m.metrics.AddProcessedMessage(xatuEvent, networkStr)

		return m.handleHandleMetadataEvent(ctx, clientMeta, traceMeta, event)

	case xatu.Event_LIBP2P_TRACE_HANDLE_STATUS.String():
		if !m.Config.Events.HandleStatusEnabled {
			return nil
		}

		// Check if we should process this event based on trace/sharding config.
		if msgID != "" && !m.ShouldTraceMessage(msgID, xatuEvent, networkStr) {
			m.metrics.AddSkippedMessage(xatuEvent, networkStr)

			return nil
		}

		m.metrics.AddProcessedMessage(xatuEvent, networkStr)

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

func mapRPCEventToXatuEvent(event string) (string, error) {
	switch event {
	case TraceEvent_HANDLE_METADATA:
		return xatu.Event_LIBP2P_TRACE_HANDLE_METADATA.String(), nil
	case TraceEvent_HANDLE_STATUS:
		return xatu.Event_LIBP2P_TRACE_HANDLE_STATUS.String(), nil
	}

	return "", errors.New("unknown libp2p rpc event")
}
