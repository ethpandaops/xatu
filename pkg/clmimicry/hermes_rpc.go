// Credit: github.com/probe-lab/hermes
// Source: hermes/host/rpc.go
// These types were extracted from the Hermes P2P network monitoring tool.

package clmimicry

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

// RpcMeta contains metadata for RPC messages.
// Extracted from github.com/probe-lab/hermes/host.
//
//nolint:tagliatelle // JSON tags match Hermes format for compatibility
type RpcMeta struct {
	PeerID        peer.ID
	Subscriptions []RpcMetaSub    `json:"Subs,omitempty"`
	Messages      []RpcMetaMsg    `json:"Msgs,omitempty"`
	Control       *RpcMetaControl `json:"Control,omitempty"`
}

// RpcMetaSub represents a subscription message.
type RpcMetaSub struct {
	Subscribe bool
	TopicID   string
}

// RpcMetaMsg represents a message in an RPC.
//
//nolint:tagliatelle // JSON tags match Hermes format for compatibility
type RpcMetaMsg struct {
	MsgID string `json:"MsgID,omitempty"`
	Topic string `json:"Topic,omitempty"`
}

// RpcMetaControl contains control messages for GossipSub.
//
//nolint:tagliatelle // JSON tags match Hermes format for compatibility
type RpcMetaControl struct {
	IHave     []RpcControlIHave     `json:"IHave,omitempty"`
	IWant     []RpcControlIWant     `json:"IWant,omitempty"`
	Graft     []RpcControlGraft     `json:"Graft,omitempty"`
	Prune     []RpcControlPrune     `json:"Prune,omitempty"`
	Idontwant []RpcControlIdontWant `json:"Idontwant,omitempty"`
}

// RpcControlIHave represents an IHAVE control message.
type RpcControlIHave struct {
	TopicID string
	MsgIDs  []string
}

// RpcControlIWant represents an IWANT control message.
type RpcControlIWant struct {
	MsgIDs []string
}

// RpcControlGraft represents a GRAFT control message.
type RpcControlGraft struct {
	TopicID string
}

// RpcControlPrune represents a PRUNE control message.
type RpcControlPrune struct {
	TopicID string
	PeerIDs []peer.ID
}

// RpcControlIdontWant represents an IDONTWANT control message.
type RpcControlIdontWant struct {
	MsgIDs []string
}
