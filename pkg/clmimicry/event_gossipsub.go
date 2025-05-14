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
	if event.Type != "HANDLE_MESSAGE" {
		return nil
	}

	// We route based on the topic of the message
	topic := event.Topic
	if topic == "" {
		return errors.New("missing topic in HandleMessage event")
	}

	// Extract MsgID for sampling decision
	msgID := getMsgID(event.Payload)

	switch {
	case strings.Contains(topic, p2p.GossipAttestationMessage):
		if !m.Config.Events.GossipSubAttestation.Enabled {
			return nil
		}

		evtName := p2p.GossipAttestationMessage

		// Check if we should process this message based on sampling config
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(evtName)

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
		if !m.Config.Events.GossipSubBeaconBlock.Enabled {
			return nil
		}

		evtName := p2p.GossipBlockMessage

		// Check if we should process this message based on sampling config
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(evtName)

		if err := m.handleGossipBeaconBlock(ctx, clientMeta, event, event.Payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub beacon block")
		}

	case strings.Contains(topic, p2p.GossipBlobSidecarMessage):
		if !m.Config.Events.GossipSubBlobSidecar.Enabled {
			return nil
		}

		evtName := p2p.GossipBlobSidecarMessage

		// Check if we should process this message based on sampling config
		if msgID != "" && !m.ShouldSampleMessage(msgID, evtName) {
			m.metrics.AddSkippedMessage(evtName)
			return nil
		}

		// Count processed message
		m.metrics.AddProcessedMessage(evtName)

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
