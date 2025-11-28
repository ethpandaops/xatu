package clmimicry

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Define events not supplied by libp2p proto pkgs.
const (
	// unknown is used as a fallback value when network ID cannot be determined.
	unknown = "unknown"

	// libp2p pubsub events.
	TraceEvent_HANDLE_MESSAGE = "HANDLE_MESSAGE"

	// libp2p core networking events.
	TraceEvent_CONNECTED    = "CONNECTED"
	TraceEvent_DISCONNECTED = "DISCONNECTED"

	// RPC events.
	TraceEvent_HANDLE_METADATA = "HANDLE_METADATA"
	TraceEvent_HANDLE_STATUS   = "HANDLE_STATUS"

	// Events that are not part of a normal Ethereum node.
	TraceEvent_SYNTHETIC_HEARTBEAT = "SYNTHETIC_HEARTBEAT"
	TraceEvent_CUSTODY_PROBE       = "CUSTODY_PROBE"
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
func (p *Processor) HandleHermesEvent(ctx context.Context, event TraceEvent) error {
	if event == nil {
		return errors.New("event is nil")
	}

	p.log.WithField("type", fmt.Sprintf("%T", event)).Trace("Received Hermes event")

	traceMeta := &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String(event.GetPeerID().String()),
	}

	clientMeta, err := p.metaProvider.GetClientMeta(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client meta: %w", err)
	}

	// Route the event to the appropriate handler based on its type.
	switch e := event.(type) {
	// GossipSub events
	case *BeaconBlockEvent:
		return p.handleBeaconBlockEvent(ctx, e, clientMeta, traceMeta)
	case *AttestationEvent:
		return p.handleAttestationEvent(ctx, e, clientMeta, traceMeta)
	case *AggregateAndProofEvent:
		return p.handleAggregateAndProofEvent(ctx, e, clientMeta, traceMeta)
	case *BlobSidecarEvent:
		return p.handleBlobSidecarEvent(ctx, e, clientMeta, traceMeta)
	case *DataColumnSidecarEvent:
		return p.handleDataColumnSidecarEvent(ctx, e, clientMeta, traceMeta)

	// libp2p trace events
	case *AddPeerEvent:
		return p.handleAddPeerEvent(ctx, e, clientMeta, traceMeta)
	case *RemovePeerEvent:
		return p.handleRemovePeerEvent(ctx, e, clientMeta, traceMeta)
	case *JoinEvent:
		return p.handleJoinEvent(ctx, e, clientMeta, traceMeta)
	case *LeaveEvent:
		return p.handleLeaveEvent(ctx, e, clientMeta, traceMeta)
	case *GraftEvent:
		return p.handleGraftEvent(ctx, e, clientMeta, traceMeta)
	case *PruneEvent:
		return p.handlePruneEvent(ctx, e, clientMeta, traceMeta)
	case *PublishMessageEvent:
		return p.handlePublishMessageEvent(ctx, e, clientMeta, traceMeta)
	case *RejectMessageEvent:
		return p.handleRejectMessageEvent(ctx, e, clientMeta, traceMeta)
	case *DuplicateMessageEvent:
		return p.handleDuplicateMessageEvent(ctx, e, clientMeta, traceMeta)
	case *DeliverMessageEvent:
		return p.handleDeliverMessageEvent(ctx, e, clientMeta, traceMeta)
	case *RecvRPCEvent:
		return p.handleRecvRPCEvent(ctx, e, clientMeta, traceMeta)
	case *SendRPCEvent:
		return p.handleSendRPCEvent(ctx, e, clientMeta, traceMeta)
	case *DropRPCEvent:
		return p.handleDropRPCEvent(ctx, e, clientMeta, traceMeta)

	// libp2p core events
	case *ConnectedEvent:
		return p.handleConnectedEvent(ctx, e, clientMeta, traceMeta)
	case *DisconnectedEvent:
		return p.handleDisconnectedEvent(ctx, e, clientMeta, traceMeta)
	case *SyntheticHeartbeatEvent:
		return p.handleSyntheticHeartbeatEvent(ctx, e, clientMeta, traceMeta)

	// RPC events
	case *HandleMetadataEvent:
		return p.handleMetadataEvent(ctx, e, clientMeta, traceMeta)
	case *HandleStatusEvent:
		return p.handleStatusEvent(ctx, e, clientMeta, traceMeta)
	case *CustodyProbeEvent:
		return p.handleCustodyProbeEvent(ctx, e, clientMeta, traceMeta)

	default:
		p.log.WithField("type", fmt.Sprintf("%T", event)).Debug("unsupported event type")

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
