package clmimicry

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/p2p"
	"github.com/pkg/errors"
	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// handleHermesGossipSubEvent handles GossipSub protocol events.
// This includes HANDLE_MESSAGE events which are further categorized by topic.
func (m *Mimicry) handleHermesGossipSubEvent(
	ctx context.Context,
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if event.Type != TraceEvent_HANDLE_MESSAGE {
		return nil
	}

	// We route based on the topic of the message
	topic := event.Topic
	if topic == "" {
		return errors.New("missing topic in handleHermesGossipSubEvent event")
	}

	// Map gossipsub event to Xatu event.
	xatuEvent, err := mapGossipSubEventToXatuEvent(topic)
	if err != nil {
		return errors.Wrap(err, "failed to map gossipsub event to xatu event")
	}

	// Extract MsgID for sharding decision.
	msgID := getMsgID(event.Payload)

	// Extract network from clientMeta
	network := clientMeta.GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = unknown
	}

	switch xatuEvent {
	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION.String():
		if !m.Config.Events.GossipSubAttestationEnabled {
			return nil
		}

		// Check if we should process this message based on trace/sharding config.
		if msgID != "" && !m.ShouldTraceMessage(msgID, xatuEvent, networkStr) {
			m.metrics.AddSkippedMessage(xatuEvent, networkStr)

			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(xatuEvent, networkStr)

		switch payload := event.Payload.(type) {
		case *eth.TraceEventAttestation:
			if err := m.handleGossipAttestation(ctx, clientMeta, event, payload); err != nil {
				return errors.Wrap(err, "failed to handle gossipsub beacon attestation")
			}
		case *eth.TraceEventSingleAttestation:
			if err := m.handleGossipSingleAttestation(ctx, clientMeta, event, payload); err != nil {
				return errors.Wrap(err, "failed to handle gossipsub single beacon attestation")
			}
		default:
			return fmt.Errorf("invalid payload type for HandleMessage event: %T", event.Payload)
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK.String():
		if !m.Config.Events.GossipSubBeaconBlockEnabled {
			return nil
		}

		// Check if we should process this message based on trace/sharding config.
		if msgID != "" && !m.ShouldTraceMessage(msgID, xatuEvent, networkStr) {
			m.metrics.AddSkippedMessage(xatuEvent, networkStr)

			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(xatuEvent, networkStr)

		if err := m.handleGossipBeaconBlock(ctx, clientMeta, event, event.Payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub beacon block")
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR.String():
		if !m.Config.Events.GossipSubBlobSidecarEnabled {
			return nil
		}

		// Check if we should process this message based on trace/sharding config.
		if msgID != "" && !m.ShouldTraceMessage(msgID, xatuEvent, networkStr) {
			m.metrics.AddSkippedMessage(xatuEvent, networkStr)

			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(xatuEvent, networkStr)

		payload, ok := event.Payload.(*eth.TraceEventBlobSidecar)
		if !ok {
			return errors.New("invalid payload type for HandleMessage event")
		}

		if err := m.handleGossipBlobSidecar(ctx, clientMeta, event, payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub blob sidecar")
		}

	default:
		m.log.WithField("topic", topic).Trace("Unsupported topic in HandleMessage event")
	}

	return nil
}

func mapGossipSubEventToXatuEvent(event string) (string, error) {
	switch event {
	case p2p.GossipAttestationMessage:
		return xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION.String(), nil
	case p2p.GossipBlockMessage:
		return xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK.String(), nil
	case p2p.GossipBlobSidecarMessage:
		return xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR.String(), nil
	}

	return "", errors.New("unknown gossipsub event")
}
