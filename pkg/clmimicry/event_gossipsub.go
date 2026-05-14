package clmimicry

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Gossip topic message type constants.
// These match the values from github.com/OffchainLabs/prysm/v7/beacon-chain/p2p/topics.go
// but are defined locally to avoid import cycles.
const (
	gossipAttestationMessage         = "beacon_attestation"
	gossipBlockMessage               = "beacon_block"
	gossipAggregateAndProofMessage   = "beacon_aggregate_and_proof"
	gossipBlobSidecarMessage         = "blob_sidecar"
	gossipDataColumnSidecarMessage   = "data_column_sidecar"
	gossipExecutionPayloadMessage    = "execution_payload"
	gossipExecutionPayloadBidMessage = "execution_payload_bid"
	gossipPayloadAttestationMessage  = "payload_attestation_message"
	gossipProposerPreferencesMessage = "proposer_preferences"
)

// Define a slice of all gossipsub event types.
var gossipsubEventTypes = []string{
	TraceEvent_HANDLE_MESSAGE,
}

// Map of gossipsub topic-name segment to Xatu event type.
// Resolution rule (see mapGossipSubEventToXatuEvent): exact match wins; otherwise
// the longest "<key>_<subnet>" prefix match wins. Exact match priority is what
// keeps `execution_payload` and `execution_payload_bid` from colliding now that
// both are valid Gloas topics.
var gossipsubTopicToXatuEventMap = map[string]string{
	gossipAttestationMessage:         xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION.String(),
	gossipBlockMessage:               xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK.String(),
	gossipBlobSidecarMessage:         xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR.String(),
	gossipAggregateAndProofMessage:   xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF.String(),
	gossipDataColumnSidecarMessage:   xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR.String(),
	gossipExecutionPayloadMessage:    xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_ENVELOPE.String(),
	gossipExecutionPayloadBidMessage: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_BID.String(),
	gossipPayloadAttestationMessage:  xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PAYLOAD_ATTESTATION_MESSAGE.String(),
	gossipProposerPreferencesMessage: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PROPOSER_PREFERENCES.String(),
}

// handleHermesGossipSubEvent handles GossipSub protocol events.
// This includes HANDLE_MESSAGE events which are further categorized by topic.
func (p *Processor) handleHermesGossipSubEvent(
	ctx context.Context,
	event *TraceEvent,
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
		p.log.WithField("topic", topic).WithContext(ctx).Tracef("unsupported topic in handleHermesGossipSubEvent event")

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
		case *TraceEventAttestation:
			if err := p.handleGossipAttestation(ctx, clientMeta, event, payload); err != nil {
				return errors.Wrap(err, "failed to handle gossipsub beacon attestation")
			}
		case *TraceEventSingleAttestation:
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

		payload, ok := event.Payload.(*TraceEventBlobSidecar)
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

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR.String():
		if !p.events.GossipSubDataColumnSidecarEnabled {
			return nil
		}

		// Record that we received this event
		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		// Check if we should process this message based on trace/sharding config.
		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		payload, ok := event.Payload.(*TraceEventDataColumnSidecar)
		if !ok {
			return errors.New("invalid payload type for HandleMessage event")
		}

		if err := p.handleGossipDataColumnSidecar(ctx, clientMeta, event, payload); err != nil {
			return errors.Wrap(err, "failed to handle data column sidecar")
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_ENVELOPE.String():
		if !p.events.GossipSubExecutionPayloadEnvelopeEnabled {
			return nil
		}

		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		payload, ok := event.Payload.(*TraceEventExecutionPayloadEnvelope)
		if !ok {
			return errors.New("invalid payload type for HandleMessage event")
		}

		if err := p.handleGossipExecutionPayloadEnvelope(ctx, clientMeta, event, payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub execution payload envelope")
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_BID.String():
		if !p.events.GossipSubExecutionPayloadBidEnabled {
			return nil
		}

		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		payload, ok := event.Payload.(*TraceEventExecutionPayloadBid)
		if !ok {
			return errors.New("invalid payload type for HandleMessage event")
		}

		if err := p.handleGossipExecutionPayloadBid(ctx, clientMeta, event, payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub execution payload bid")
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PAYLOAD_ATTESTATION_MESSAGE.String():
		if !p.events.GossipSubPayloadAttestationMessageEnabled {
			return nil
		}

		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		payload, ok := event.Payload.(*TraceEventPayloadAttestationMessage)
		if !ok {
			return errors.New("invalid payload type for HandleMessage event")
		}

		if err := p.handleGossipPayloadAttestationMessage(ctx, clientMeta, event, payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub payload attestation message")
		}

	case xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PROPOSER_PREFERENCES.String():
		if !p.events.GossipSubProposerPreferencesEnabled {
			return nil
		}

		networkStr := getNetworkID(clientMeta)
		p.metrics.AddEvent(xatuEvent, networkStr)

		if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
			return nil
		}

		payload, ok := event.Payload.(*TraceEventProposerPreferences)
		if !ok {
			return errors.New("invalid payload type for HandleMessage event")
		}

		if err := p.handleGossipProposerPreferences(ctx, clientMeta, event, payload); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub proposer preferences")
		}

	default:
		p.log.WithField("topic", topic).WithContext(ctx).Trace("Unsupported topic in HandleMessage event")
	}

	return nil
}

// mapGossipSubEventToXatuEvent resolves a gossipsub topic string to a xatu
// event name. The topic shape is "/eth2/<fork_digest>/<topic_name>/<encoding>";
// we match against the <topic_name> segment.
//
// Exact match on the segment wins. If no exact match, the longest matching
// "<key>_" prefix wins — this handles subnet-suffixed topics like
// "beacon_attestation_5" / "blob_sidecar_0" / "data_column_sidecar_5".
// Map iteration is non-deterministic, so the "longest-prefix wins" pass is
// what prevents a `execution_payload_bid` topic from being routed to the
// `execution_payload` envelope branch (or vice versa).
func mapGossipSubEventToXatuEvent(topic string) (string, error) {
	parts := strings.Split(topic, "/")
	if len(parts) < 4 {
		return "", fmt.Errorf("malformed gossipsub topic: %s", topic)
	}

	segment := parts[3]

	if xatuEvent, ok := gossipsubTopicToXatuEventMap[segment]; ok {
		return xatuEvent, nil
	}

	var (
		bestKey   string
		bestEvent string
	)

	for topicKey, xatuEvent := range gossipsubTopicToXatuEventMap {
		if strings.HasPrefix(segment, topicKey+"_") && len(topicKey) > len(bestKey) {
			bestKey = topicKey
			bestEvent = xatuEvent
		}
	}

	if bestEvent != "" {
		return bestEvent, nil
	}

	return "", fmt.Errorf("unknown gossipsub event: %s", topic)
}
