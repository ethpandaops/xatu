package clmimicry

import (
	"context"
	"fmt"
	"strings"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/p2p"
	"github.com/pkg/errors"
	"github.com/probe-lab/hermes/eth/events"
	"github.com/probe-lab/hermes/host"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Define a slice of all gossipsub event types.
var gossipsubEventTypes = []string{
	TraceEvent_HANDLE_MESSAGE,
}

// Map of gossipsub topic substrings to Xatu event types.
var gossipsubTopicToXatuEventMap = map[string]string{
	p2p.GossipAttestationMessage:       xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION.String(),
	p2p.GossipBlockMessage:             xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK.String(),
	p2p.GossipBlobSidecarMessage:       xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR.String(),
	p2p.GossipAggregateAndProofMessage: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF.String(),
}

// handleHermesGossipSubEvent handles GossipSub protocol events.
// This includes HANDLE_MESSAGE events which are further categorized by topic.
func (p *Processor) handleHermesGossipSubEvent(
	ctx context.Context,
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if event.Type != TraceEvent_HANDLE_MESSAGE {
		return nil
	}

	// We route based on the topic of the message.
	topic := event.Topic
	if topic == "" {
		return errors.New("missing topic in handleHermesGossipSubEvent event")
	}

	// Map gossipsub event to Xatu event.
	xatuEvent, err := mapGossipSubEventToXatuEvent(topic)
	if err != nil {
		p.log.WithField("topic", topic).Tracef("unsupported topic in handleHermesGossipSubEvent event")

		//nolint:nilerr // we don't want to return an error here.
		return nil
	}

	switch xatuEvent {
	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION.String():
		if !p.events.GossipSubAttestationEnabled {
			return nil
		}

		// Record that we received this event
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this message based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		switch payload := event.Payload.(type) {
		case *events.TraceEventAttestation:
			if err := p.handleGossipAttestation(ctx, clientMeta, event, payload); err != nil {
				return errors.Wrap(err, "failed to handle gossipsub beacon attestation")
			}
		case *events.TraceEventSingleAttestation:
			if err := p.handleGossipSingleAttestation(ctx, clientMeta, event, payload); err != nil {
				return errors.Wrap(err, "failed to handle gossipsub single beacon attestation")
			}
		default:
			return fmt.Errorf("invalid payload type for HandleMessage event: %T", event.Payload)
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK.String():
		if !p.events.GossipSubBeaconBlockEnabled {
			return nil
		}

		// Record that we received this event
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this message based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		if err := p.handleGossipBeaconBlock(ctx, clientMeta, event, event.Payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub beacon block")
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR.String():
		if !p.events.GossipSubBlobSidecarEnabled {
			return nil
		}

		// Record that we received this event
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this message based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		payload, ok := event.Payload.(*events.TraceEventBlobSidecar)
		if !ok {
			return errors.New("invalid payload type for HandleMessage event")
		}

		if err := p.handleGossipBlobSidecar(ctx, clientMeta, event, payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub blob sidecar")
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF.String():
		if !p.events.GossipSubAggregateAndProofEnabled {
			return nil
		}

		// Record that we received this event
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this message based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		if err := p.handleGossipAggregateAndProof(ctx, clientMeta, event, event.Payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub aggregate and proof")
		}

	default:
		p.log.WithField("topic", topic).Trace("Unsupported topic in HandleMessage event")
	}

	return nil
}

func mapGossipSubEventToXatuEvent(topic string) (string, error) {
	for topicSubstr, xatuEvent := range gossipsubTopicToXatuEventMap {
		if strings.Contains(topic, topicSubstr) {
			return xatuEvent, nil
		}
	}

	return "", fmt.Errorf("unknown gossipsub event: %s", topic)
}
