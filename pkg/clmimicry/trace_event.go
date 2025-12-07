package clmimicry

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TraceEvent is the interface for all trace events from the consensus layer P2P network.
// Each concrete event type embeds TraceEventBase and adds its specific fields.
type TraceEvent interface {
	// GetTimestamp returns when the event occurred.
	GetTimestamp() time.Time
	// GetPeerID returns the peer ID associated with this event.
	GetPeerID() peer.ID
}

// TraceEventBase contains common fields for all trace events.
// Embed this in concrete event types to satisfy the TraceEvent interface.
type TraceEventBase struct {
	Timestamp time.Time
	PeerID    peer.ID
}

// GetTimestamp returns when the event occurred.
func (b TraceEventBase) GetTimestamp() time.Time {
	return b.Timestamp
}

// GetPeerID returns the peer ID associated with this event.
func (b TraceEventBase) GetPeerID() peer.ID {
	return b.PeerID
}

// TopicEvent is an optional interface for events that have a topic.
type TopicEvent interface {
	TraceEvent
	GetTopic() string
}

// MessageEvent is an optional interface for events that have a message ID.
type MessageEvent interface {
	TraceEvent
	GetMsgID() string
}

// RPCMetaEvent is an optional interface for RPC events that contain RpcMeta.
type RPCMetaEvent interface {
	TraceEvent
	GetRPCMeta() *RpcMeta
}
