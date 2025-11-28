package clmimicry

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// ConnectedEvent represents a libp2p peer connection event.
// This event is emitted when a connection to a remote peer is established.
type ConnectedEvent struct {
	TraceEventBase
	RemotePeer   string
	RemoteMaddrs ma.Multiaddr
	AgentVersion string
	Direction    string
	Opened       time.Time
	Limited      bool
}

// Verify interface compliance at compile time
var _ TraceEvent = (*ConnectedEvent)(nil)

// DisconnectedEvent represents a libp2p peer disconnection event.
// This event is emitted when a connection to a remote peer is closed.
type DisconnectedEvent struct {
	TraceEventBase
	RemotePeer   string
	RemoteMaddrs ma.Multiaddr
	AgentVersion string
	Direction    string
	Opened       time.Time
	Limited      bool
}

// Verify interface compliance at compile time
var _ TraceEvent = (*DisconnectedEvent)(nil)

// SyntheticHeartbeatEvent represents a synthetic heartbeat event for active connections.
// This event is periodically generated to track the health and status of peer connections.
type SyntheticHeartbeatEvent struct {
	TraceEventBase
	RemotePeer      string
	RemoteMaddrs    ma.Multiaddr
	LatencyMs       int64
	AgentVersion    string
	Direction       uint32
	Protocols       []string
	ConnectionAgeNs int64
}

// Verify interface compliance at compile time
var _ TraceEvent = (*SyntheticHeartbeatEvent)(nil)

// NewConnectedEvent creates a new ConnectedEvent with the provided fields.
func NewConnectedEvent(
	timestamp time.Time,
	peerID peer.ID,
	remotePeer string,
	remoteMaddrs ma.Multiaddr,
	agentVersion string,
	direction string,
	opened time.Time,
	limited bool,
) *ConnectedEvent {
	return &ConnectedEvent{
		TraceEventBase: TraceEventBase{
			Timestamp: timestamp,
			PeerID:    peerID,
		},
		RemotePeer:   remotePeer,
		RemoteMaddrs: remoteMaddrs,
		AgentVersion: agentVersion,
		Direction:    direction,
		Opened:       opened,
		Limited:      limited,
	}
}

// NewDisconnectedEvent creates a new DisconnectedEvent with the provided fields.
func NewDisconnectedEvent(
	timestamp time.Time,
	peerID peer.ID,
	remotePeer string,
	remoteMaddrs ma.Multiaddr,
	agentVersion string,
	direction string,
	opened time.Time,
	limited bool,
) *DisconnectedEvent {
	return &DisconnectedEvent{
		TraceEventBase: TraceEventBase{
			Timestamp: timestamp,
			PeerID:    peerID,
		},
		RemotePeer:   remotePeer,
		RemoteMaddrs: remoteMaddrs,
		AgentVersion: agentVersion,
		Direction:    direction,
		Opened:       opened,
		Limited:      limited,
	}
}

// NewSyntheticHeartbeatEvent creates a new SyntheticHeartbeatEvent with the provided fields.
func NewSyntheticHeartbeatEvent(
	timestamp time.Time,
	peerID peer.ID,
	remotePeer string,
	remoteMaddrs ma.Multiaddr,
	latencyMs int64,
	agentVersion string,
	direction uint32,
	protocols []string,
	connectionAgeNs int64,
) *SyntheticHeartbeatEvent {
	return &SyntheticHeartbeatEvent{
		TraceEventBase: TraceEventBase{
			Timestamp: timestamp,
			PeerID:    peerID,
		},
		RemotePeer:      remotePeer,
		RemoteMaddrs:    remoteMaddrs,
		LatencyMs:       latencyMs,
		AgentVersion:    agentVersion,
		Direction:       direction,
		Protocols:       protocols,
		ConnectionAgeNs: connectionAgeNs,
	}
}
