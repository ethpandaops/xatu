package clmimicry

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// handleBeaconBlockEvent handles beacon block gossipsub events.
func (p *Processor) handleBeaconBlockEvent(
	ctx context.Context,
	event *BeaconBlockEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.GossipSubBeaconBlockEnabled {
		return nil
	}

	// Record that we received this event
	networkStr := getNetworkID(clientMeta)
	xatuEvent := xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK.String()
	p.metrics.AddEvent(xatuEvent, networkStr)

	// Check if we should process this message based on trace/sharding config.
	if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
		return nil
	}

	if err := p.handleGossipBeaconBlock(ctx, event, clientMeta, traceMeta); err != nil {
		return errors.Wrap(err, "failed to handle gossipsub beacon block")
	}

	return nil
}

// handleAttestationEvent handles attestation gossipsub events.
func (p *Processor) handleAttestationEvent(
	ctx context.Context,
	event *AttestationEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.GossipSubAttestationEnabled {
		return nil
	}

	// Record that we received this event
	networkStr := getNetworkID(clientMeta)
	xatuEvent := xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION.String()
	p.metrics.AddEvent(xatuEvent, networkStr)

	// Check if we should process this message based on trace/sharding config.
	if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
		return nil
	}

	switch event.Payload.(type) {
	case *TraceEventAttestation:
		if err := p.handleGossipAttestation(ctx, event, clientMeta, traceMeta); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub beacon attestation")
		}
	case *TraceEventSingleAttestation:
		if err := p.handleGossipSingleAttestation(ctx, event, clientMeta, traceMeta); err != nil {
			return errors.Wrap(err, "failed to handle gossipsub single beacon attestation")
		}
	default:
		return fmt.Errorf("invalid payload type for attestation event: %T", event.Payload)
	}

	return nil
}

// handleAggregateAndProofEvent handles aggregate and proof gossipsub events.
func (p *Processor) handleAggregateAndProofEvent(
	ctx context.Context,
	event *AggregateAndProofEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.GossipSubAggregateAndProofEnabled {
		return nil
	}

	// Record that we received this event
	networkStr := getNetworkID(clientMeta)
	xatuEvent := xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF.String()
	p.metrics.AddEvent(xatuEvent, networkStr)

	// Check if we should process this message based on trace/sharding config.
	if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
		return nil
	}

	if err := p.handleGossipAggregateAndProof(ctx, event, clientMeta, traceMeta); err != nil {
		return errors.Wrap(err, "failed to handle gossipsub aggregate and proof")
	}

	return nil
}

// handleBlobSidecarEvent handles blob sidecar gossipsub events.
func (p *Processor) handleBlobSidecarEvent(
	ctx context.Context,
	event *BlobSidecarEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.GossipSubBlobSidecarEnabled {
		return nil
	}

	// Record that we received this event
	networkStr := getNetworkID(clientMeta)
	xatuEvent := xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR.String()
	p.metrics.AddEvent(xatuEvent, networkStr)

	// Check if we should process this message based on trace/sharding config.
	if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
		return nil
	}

	if err := p.handleGossipBlobSidecar(ctx, event, clientMeta, traceMeta); err != nil {
		return errors.Wrap(err, "failed to handle gossipsub blob sidecar")
	}

	return nil
}

// handleDataColumnSidecarEvent handles data column sidecar gossipsub events.
func (p *Processor) handleDataColumnSidecarEvent(
	ctx context.Context,
	event *DataColumnSidecarEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	if !p.events.GossipSubDataColumnSidecarEnabled {
		return nil
	}

	// Record that we received this event
	networkStr := getNetworkID(clientMeta)
	xatuEvent := xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR.String()
	p.metrics.AddEvent(xatuEvent, networkStr)

	// Check if we should process this message based on trace/sharding config.
	if !p.ShouldTraceMessage(event, clientMeta, xatuEvent) {
		return nil
	}

	if err := p.handleGossipDataColumnSidecar(ctx, event, clientMeta, traceMeta); err != nil {
		return errors.Wrap(err, "failed to handle data column sidecar")
	}

	return nil
}
