package clmimicry

import (
	"github.com/libp2p/go-libp2p/core/protocol"
)

// AddPeerEvent represents an event when a peer is added to the libp2p network.
type AddPeerEvent struct {
	TraceEventBase
	Protocol protocol.ID
}

// RemovePeerEvent represents an event when a peer is removed from the libp2p network.
type RemovePeerEvent struct {
	TraceEventBase
}

// JoinEvent represents an event when the local node joins a GossipSub topic.
type JoinEvent struct {
	TraceEventBase
	Topic string
}

// GetTopic returns the topic that was joined.
func (e JoinEvent) GetTopic() string {
	return e.Topic
}

// LeaveEvent represents an event when the local node leaves a GossipSub topic.
type LeaveEvent struct {
	TraceEventBase
	Topic string
}

// GetTopic returns the topic that was left.
func (e LeaveEvent) GetTopic() string {
	return e.Topic
}

// GraftEvent represents a GossipSub GRAFT event.
type GraftEvent struct {
	TraceEventBase
	Topic string
}

// GetTopic returns the topic for the graft operation.
func (e GraftEvent) GetTopic() string {
	return e.Topic
}

// PruneEvent represents a GossipSub PRUNE event.
type PruneEvent struct {
	TraceEventBase
	Topic string
}

// GetTopic returns the topic for the prune operation.
func (e PruneEvent) GetTopic() string {
	return e.Topic
}

// PublishMessageEvent represents an event when a message is published to GossipSub.
type PublishMessageEvent struct {
	TraceEventBase
	MsgID string
	Topic string
}

// GetMsgID returns the message ID.
func (e PublishMessageEvent) GetMsgID() string {
	return e.MsgID
}

// GetTopic returns the topic the message was published to.
func (e PublishMessageEvent) GetTopic() string {
	return e.Topic
}

// RejectMessageEvent represents an event when a GossipSub message is rejected.
type RejectMessageEvent struct {
	TraceEventBase
	MsgID     string
	Topic     string
	Reason    string
	Local     bool
	MsgSize   int
	SeqNumber uint64
}

// GetMsgID returns the message ID.
func (e *RejectMessageEvent) GetMsgID() string {
	return e.MsgID
}

// GetTopic returns the topic the message was rejected from.
func (e *RejectMessageEvent) GetTopic() string {
	return e.Topic
}

// DuplicateMessageEvent represents an event when a duplicate GossipSub message is received.
type DuplicateMessageEvent struct {
	TraceEventBase
	MsgID     string
	Topic     string
	Local     bool
	MsgSize   int
	SeqNumber uint64
}

// GetMsgID returns the message ID.
func (e *DuplicateMessageEvent) GetMsgID() string {
	return e.MsgID
}

// GetTopic returns the topic the duplicate message was received on.
func (e *DuplicateMessageEvent) GetTopic() string {
	return e.Topic
}

// DeliverMessageEvent represents an event when a GossipSub message is delivered.
type DeliverMessageEvent struct {
	TraceEventBase
	MsgID     string
	Topic     string
	Local     bool
	MsgSize   int
	SeqNumber uint64
}

// GetMsgID returns the message ID.
func (e *DeliverMessageEvent) GetMsgID() string {
	return e.MsgID
}

// GetTopic returns the topic the message was delivered on.
func (e *DeliverMessageEvent) GetTopic() string {
	return e.Topic
}

// RecvRPCEvent represents an event when an RPC message is received.
type RecvRPCEvent struct {
	TraceEventBase
	Meta *RpcMeta
}

// GetRPCMeta returns the RPC metadata.
func (e *RecvRPCEvent) GetRPCMeta() *RpcMeta {
	return e.Meta
}

// SendRPCEvent represents an event when an RPC message is sent.
type SendRPCEvent struct {
	TraceEventBase
	Meta *RpcMeta
}

// GetRPCMeta returns the RPC metadata.
func (e *SendRPCEvent) GetRPCMeta() *RpcMeta {
	return e.Meta
}

// DropRPCEvent represents an event when an RPC message is dropped.
type DropRPCEvent struct {
	TraceEventBase
	Meta *RpcMeta
}

// GetRPCMeta returns the RPC metadata.
func (e *DropRPCEvent) GetRPCMeta() *RpcMeta {
	return e.Meta
}

// Compile-time interface compliance checks
var (
	_ TraceEvent   = (*AddPeerEvent)(nil)
	_ TraceEvent   = (*RemovePeerEvent)(nil)
	_ TraceEvent   = (*JoinEvent)(nil)
	_ TopicEvent   = (*JoinEvent)(nil)
	_ TraceEvent   = (*LeaveEvent)(nil)
	_ TopicEvent   = (*LeaveEvent)(nil)
	_ TraceEvent   = (*GraftEvent)(nil)
	_ TopicEvent   = (*GraftEvent)(nil)
	_ TraceEvent   = (*PruneEvent)(nil)
	_ TopicEvent   = (*PruneEvent)(nil)
	_ TraceEvent   = (*PublishMessageEvent)(nil)
	_ MessageEvent = (*PublishMessageEvent)(nil)
	_ TopicEvent   = (*PublishMessageEvent)(nil)
	_ TraceEvent   = (*RejectMessageEvent)(nil)
	_ MessageEvent = (*RejectMessageEvent)(nil)
	_ TopicEvent   = (*RejectMessageEvent)(nil)
	_ TraceEvent   = (*DuplicateMessageEvent)(nil)
	_ MessageEvent = (*DuplicateMessageEvent)(nil)
	_ TopicEvent   = (*DuplicateMessageEvent)(nil)
	_ TraceEvent   = (*DeliverMessageEvent)(nil)
	_ MessageEvent = (*DeliverMessageEvent)(nil)
	_ TopicEvent   = (*DeliverMessageEvent)(nil)
	_ TraceEvent   = (*RecvRPCEvent)(nil)
	_ RPCMetaEvent = (*RecvRPCEvent)(nil)
	_ TraceEvent   = (*SendRPCEvent)(nil)
	_ RPCMetaEvent = (*SendRPCEvent)(nil)
	_ TraceEvent   = (*DropRPCEvent)(nil)
	_ RPCMetaEvent = (*DropRPCEvent)(nil)
)
