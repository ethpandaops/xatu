// Credit: github.com/probe-lab/hermes
// Source: hermes/host/flush_tracer.go
// These types were extracted from the Hermes P2P network monitoring tool.

package clmimicry

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TraceEvent represents a trace event from the libp2p network.
// Extracted from github.com/probe-lab/hermes/host.
//
//nolint:tagliatelle // JSON tags match Hermes format for compatibility
type TraceEvent struct {
	Type      string
	Topic     string
	PeerID    peer.ID
	Timestamp time.Time
	Payload   any `json:"Data"` // cannot use field "Data" because of gk.Record method
}

// TraceEventPayloadMetaData contains metadata for trace event payloads.
// Extracted from github.com/probe-lab/hermes/host.
//
//nolint:tagliatelle // JSON tags match Hermes format for compatibility
type TraceEventPayloadMetaData struct {
	PeerID  string `json:"PeerID"`
	Topic   string `json:"Topic"`
	Seq     []byte `json:"Seq"`
	MsgID   string `json:"MsgID"`
	MsgSize int    `json:"MsgSize"`
}
