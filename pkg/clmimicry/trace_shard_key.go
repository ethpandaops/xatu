package clmimicry

import (
	"fmt"
	"reflect"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/probe-lab/hermes/host"
)

// ShardingKeyType represents the different types of sharding keys.
type ShardingKeyType string

const (
	// ShardingKeyTypeMsgID uses the message ID for sharding.
	ShardingKeyTypeMsgID ShardingKeyType = "MsgID"
	// ShardingKeyTypePeerID uses the peer ID for sharding.
	ShardingKeyTypePeerID ShardingKeyType = "PeerID"
)

// GetShardingKey extracts the appropriate sharding key based on the configured type.
// Default to MsgID if the event type is not supported.
func GetShardingKey(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	shardingKeyType string,
	eventType string,
) string {
	switch ShardingKeyType(shardingKeyType) {
	case ShardingKeyTypePeerID:
		return getPeerID(eventType, event)
	default:
		return getMsgID(event.Payload)
	}
}

func getPeerID(eventType string, event *host.TraceEvent) string {
	switch eventType {
	case xatu.Event_LIBP2P_TRACE_CONNECTED.String():
		if c, err := libp2p.TraceEventToConnected(event); err == nil && c.RemotePeer != nil {
			return c.RemotePeer.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_DISCONNECTED.String():
		if d, err := libp2p.TraceEventToDisconnected(event); err == nil && d.RemotePeer != nil {
			return d.RemotePeer.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_HANDLE_METADATA.String():
		if m, err := libp2p.TraceEventToHandleMetadata(event); err == nil && m.PeerId != nil {
			return m.PeerId.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_HANDLE_STATUS.String():
		if s, err := libp2p.TraceEventToHandleStatus(event); err == nil && s.PeerId != nil {
			return s.PeerId.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_ADD_PEER.String():
		if a, err := libp2p.TraceEventToAddPeer(event); err == nil && a.PeerId != nil {
			return a.PeerId.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_REMOVE_PEER.String():
		if r, err := libp2p.TraceEventToRemovePeer(event); err == nil && r.PeerId != nil {
			return r.PeerId.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_RECV_RPC.String():
		if r, err := libp2p.TraceEventToRecvRPC(event); err == nil && r.PeerId != nil {
			return r.PeerId.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_SEND_RPC.String():
		if s, err := libp2p.TraceEventToSendRPC(event); err == nil && s.PeerId != nil {
			return s.PeerId.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE.String():
		if d, err := libp2p.TraceEventToDeliverMessage(event); err == nil && d.PeerId != nil {
			return d.PeerId.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE.String():
		if r, err := libp2p.TraceEventToRejectMessage(event); err == nil && r.PeerId != nil {
			return r.PeerId.GetValue()
		}
	case xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE.String():
		if d, err := libp2p.TraceEventToDuplicateMessage(event); err == nil && d.PeerId != nil {
			return d.PeerId.GetValue()
		}
	default:
	}

	// For other event types (mainly gossipsub), try the map-style access.
	if payload, ok := event.Payload.(map[string]any); ok {
		// First, check if the event payload has a PeerID field. This is the remote peer ID for
		// some hermes events.
		if remotePeerID, found := payload["PeerID"]; found {
			return fmt.Sprintf("%v", remotePeerID)
		}

		// Some events use RemotePeer as a field name, so we'll check for that too.
		if remotePeerID, found := payload["RemotePeer"]; found {
			return fmt.Sprintf("%v", remotePeerID)
		}
	}

	// We'll just default to the host peer ID if we can't extract the remote peer ID.
	// This will mean the event won't be sharded correctly, but will parse fine and
	// show in metrics.
	return event.PeerID.String()
}

// getMsgID extracts the MsgID field from any supported payload type.
// The alternative to using reflection here is a massive switch statement that we
// would need to manage. If we find CPU issues, we should switch it out, pun intended.
func getMsgID(payload interface{}) string {
	// Use reflection to access the MsgID field.
	if payload == nil {
		return ""
	}

	// Handle map[string]any payloads (used by deliver_message, duplicate_message, etc.)
	if mapPayload, ok := payload.(map[string]any); ok {
		if msgID, found := mapPayload["MsgID"]; found {
			if msgIDStr, ok := msgID.(string); ok {
				return msgIDStr
			}
		}

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
