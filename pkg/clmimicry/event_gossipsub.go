package clmimicry

import (
	"context"
	"fmt"
	"strings"

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

	// Extract MsgID for sampling decision
	msgID := getMsgID(event.Payload)

	// Extract network from clientMeta
	network := clientMeta.GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = unknown
	}

	switch {
	case strings.Contains(topic, p2p.GossipAttestationMessage):
		if !m.Config.Events.GossipSubAttestationEnabled {
			return nil
		}

		evtName := p2p.GossipAttestationMessage

		// Check if we should process this message based on sampling config
		if msgID != "" && !m.ShouldTraceMessage(msgID, evtName, networkStr) {
			m.metrics.AddSkippedMessage(evtName, networkStr)

			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(evtName, networkStr)

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

	case strings.Contains(topic, p2p.GossipBlockMessage):
		if !m.Config.Events.GossipSubBeaconBlockEnabled {
			return nil
		}

		evtName := p2p.GossipBlockMessage

		// Check if we should process this message based on sampling config
		if msgID != "" && !m.ShouldTraceMessage(msgID, evtName, networkStr) {
			m.metrics.AddSkippedMessage(evtName, networkStr)

			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(evtName, networkStr)

		if err := m.handleGossipBeaconBlock(ctx, clientMeta, event, event.Payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub beacon block")
		}

	case strings.Contains(topic, p2p.GossipBlobSidecarMessage):
		if !m.Config.Events.GossipSubBlobSidecarEnabled {
			return nil
		}

		evtName := p2p.GossipBlobSidecarMessage

		// Check if we should process this message based on sampling config
		if msgID != "" && !m.ShouldTraceMessage(msgID, evtName, networkStr) {
			m.metrics.AddSkippedMessage(evtName, networkStr)

			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(evtName, networkStr)

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
