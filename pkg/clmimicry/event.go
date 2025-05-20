package clmimicry

import (
	"context"
	"fmt"
	"slices"

	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/pkg/errors"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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

var (
	// Some events dont have anything reasonable to shard on, so we let them all through.
	UnshardableEventTypes = []string{
		xatu.Event_LIBP2P_TRACE_JOIN.String(),
		xatu.Event_LIBP2P_TRACE_LEAVE.String(),
	}
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
func (m *Mimicry) handleHermesEvent(ctx context.Context, event *host.TraceEvent) error {
	if event == nil {
		return errors.New("event is nil")
	}

	m.log.WithField("type", event.Type).Trace("Received Hermes event")

	clientMeta, err := m.createNewClientMeta(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to create new client meta")
	}

	traceMeta := &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String(event.PeerID.String()),
	}

	// Route the event to the appropriate handler based on its category.
	switch {
	// GossipSub protocol events.
	case isGossipSubEvent(event):
		return m.handleHermesGossipSubEvent(ctx, event, clientMeta, traceMeta)

	// libp2p pubsub protocol level events.
	case isLibp2pEvent(event):
		return m.handleHermesLibp2pEvent(ctx, event, clientMeta, traceMeta)

	// libp2p core networking events.
	case isLibp2pCoreEvent(event):
		return m.handleHermesLibp2pCoreEvent(ctx, event, clientMeta, traceMeta)

	// Request/Response (RPC) protocol events.
	case isRpcEvent(event):
		return m.handleHermesRPCEvent(ctx, event, clientMeta, traceMeta)

	default:
		m.log.WithField("type", event.Type).Trace("unsupported Hermes event")

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
	return event.Type == TraceEvent_HANDLE_METADATA || event.Type == TraceEvent_HANDLE_STATUS
}

// isLibp2pCoreEvent checks if the event is a libp2p core event.
func isLibp2pCoreEvent(event *host.TraceEvent) bool {
	return event.Type == TraceEvent_CONNECTED || event.Type == TraceEvent_DISCONNECTED
}

// isLibp2pEvent checks if the event is a libp2p event.
func isLibp2pEvent(event *host.TraceEvent) bool {
	return event.Type == pubsubpb.TraceEvent_ADD_PEER.String() ||
		event.Type == pubsubpb.TraceEvent_REMOVE_PEER.String() ||
		event.Type == pubsubpb.TraceEvent_RECV_RPC.String() ||
		event.Type == pubsubpb.TraceEvent_SEND_RPC.String() ||
		event.Type == pubsubpb.TraceEvent_DROP_RPC.String() ||
		event.Type == pubsubpb.TraceEvent_JOIN.String() ||
		event.Type == pubsubpb.TraceEvent_GRAFT.String() ||
		event.Type == pubsubpb.TraceEvent_PRUNE.String() ||
		event.Type == pubsubpb.TraceEvent_LEAVE.String()
}

// isGossipSubEvent checks if the event is a gossipsub event.
func isGossipSubEvent(event *host.TraceEvent) bool {
	return event.Type == TraceEvent_HANDLE_MESSAGE
}

// isUnshardableEvent checks if the event type is unshardable.
func isUnshardableEvent(eventType string) bool {
	return slices.Contains(UnshardableEventTypes, eventType)
}
