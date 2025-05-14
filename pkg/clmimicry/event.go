package clmimicry

import (
	"context"
	"reflect"

	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/pkg/errors"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
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
	case event.Type == "HANDLE_MESSAGE":
		return m.handleHermesGossipSubEvent(ctx, event, clientMeta, traceMeta)

	// libp2p pubsub protocol level events.
	case event.Type == pubsubpb.TraceEvent_ADD_PEER.String() ||
		event.Type == pubsubpb.TraceEvent_REMOVE_PEER.String() ||
		event.Type == pubsubpb.TraceEvent_RECV_RPC.String() ||
		event.Type == pubsubpb.TraceEvent_SEND_RPC.String() ||
		event.Type == pubsubpb.TraceEvent_JOIN.String():
		return m.handleHermesLibp2pEvent(ctx, event, clientMeta, traceMeta)

	// libp2p core networking events.
	case event.Type == "CONNECTED" || event.Type == "DISCONNECTED":
		return m.handleHermesLibp2pCoreEvent(ctx, event, clientMeta, traceMeta)

	// Request/Response (RPC) protocol events.
	case event.Type == "HANDLE_METADATA" || event.Type == "HANDLE_STATUS":
		return m.handleHermesRPCEvent(ctx, event, clientMeta, traceMeta)

	default:
		m.log.WithField("type", event.Type).Trace("unsupported Hermes event")

		return nil
	}
}

// getMsgID extracts the MsgID field from any supported payload type.
// The alternative to using reflection here is a massive switch statement that we
// would need to manage. If we find CPU issues, we should switch it out, pun intended.
func getMsgID(payload interface{}) string {
	// Use reflection to access the MsgID field.
	if payload == nil {
		return ""
	}

	// Try to access the MsgID field using reflection.
	v := reflect.ValueOf(payload)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ""
	}

	// Dereference the pointer and check if it's a struct.
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return ""
	}

	// Try to find the MsgID field.
	msgIDField := v.FieldByName("MsgID")
	if !msgIDField.IsValid() || msgIDField.Kind() != reflect.String {
		return ""
	}

	return msgIDField.String()
}
