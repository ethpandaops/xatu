package clmimicry

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Define events not supplied by libp2p proto pkgs.
const (
	// libp2p pubsub events.
	TraceEvent_HANDLE_MESSAGE = "HANDLE_MESSAGE"

	// libp2p core networking events.
	TraceEvent_CONNECTED    = "CONNECTED"
	TraceEvent_DISCONNECTED = "DISCONNECTED"

	// RPC events.
	TraceEvent_HANDLE_METADATA = "HANDLE_METADATA"
	TraceEvent_HANDLE_STATUS   = "HANDLE_STATUS"
)

// handleHermesEvent processes events from Hermes and routes them to appropriate handlers based on their type.
// To better understand this, here's a breakdown of the events, categorised:
//
// 1. GossipSub protocol events:
//   - "HANDLE_MESSAGE": Processes gossip messages based on their topic
//   - Attestations (p2p.GossipAttestationMessage)
//   - Beacon Blocks (p2p.GossipBlockMessage)
//   - Blob Sidecars (p2p.GossipBlobSidecarMessage)
//
// 2. libp2p pubsub protocol level events:
//   - "ADD_PEER": When a peer is added to the pubsub system
//   - "REMOVE_PEER": When a peer is removed from the pubsub system
//   - "RECV_RPC": When an RPC message is received
//   - "SEND_RPC": When an RPC message is sent
//   - "JOIN": When joining a pubsub topic
//
// 3. libp2p core networking events:
//   - "CONNECTED": When a peer connects at the network layer (from libp2p's network.Notify)
//   - "DISCONNECTED": When a peer disconnects at the network layer (from libp2p's network.Notify)
//
// 3. Request/Response (RPC) protocol events:
//   - "HANDLE_METADATA": Processing of metadata requests
//   - "HANDLE_STATUS": Processing of status requests

// HandleHermesEvent processes a Hermes trace event and routes it to the appropriate handler
func (p *Processor) HandleHermesEvent(ctx context.Context, event *host.TraceEvent) error {
	if event == nil {
		return errors.New("event is nil")
	}

	p.log.WithField("type", event.Type).Trace("Received Hermes event")

	traceMeta := &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String(event.PeerID.String()),
	}

	clientMeta, err := p.metaProvider.GetClientMeta(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client meta: %w", err)
	}

	// Route the event to the appropriate handler based on its category.
	switch {
	// GossipSub protocol events.
	case isGossipSubEvent(event):
		return p.handleHermesGossipSubEvent(ctx, event, clientMeta, traceMeta)

	// libp2p pubsub protocol level events.
	case isLibp2pEvent(event):
		return p.handleHermesLibp2pEvent(ctx, event, clientMeta, traceMeta)

	// libp2p core networking events.
	case isLibp2pCoreEvent(event):
		return p.handleHermesLibp2pCoreEvent(ctx, event, clientMeta, traceMeta)

	// Request/Response (RPC) protocol events.
	case isRpcEvent(event):
		return p.handleHermesRPCEvent(ctx, event, clientMeta, traceMeta)

	default:
		p.log.WithField("type", event.Type).Debug("unsupported Hermes event")

		return nil
	}
}

// getNetworkID extracts the network ID from the client metadata.
func getNetworkID(clientMeta *xatu.ClientMeta) string {
	var (
		network    = clientMeta.GetEthereum().GetNetwork().GetId()
		networkStr = fmt.Sprintf("%d", network)
	)

	if networkStr == "" || networkStr == "0" {
		networkStr = unknown
	}

	return networkStr
}

// isRpcEvent checks if the event is a RPC event.
func isRpcEvent(event *host.TraceEvent) bool {
	_, exists := rpcToXatuEventMap[event.Type]

	return exists
}

// isLibp2pCoreEvent checks if the event is a libp2p core event.
func isLibp2pCoreEvent(event *host.TraceEvent) bool {
	_, exists := libp2pCoreToXatuEventMap[event.Type]

	return exists
}

// isLibp2pEvent checks if the event is a libp2p event.
func isLibp2pEvent(event *host.TraceEvent) bool {
	_, exists := libp2pToXatuEventMap[event.Type]

	return exists
}

// isGossipSubEvent checks if the event is a gossipsub event.
func isGossipSubEvent(event *host.TraceEvent) bool {
	return slices.Contains(gossipsubEventTypes, event.Type)
}
