package libp2p

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Helper function to convert a Hermes TraceEvent to a libp2p AddPeer
func TraceEventToAddPeer(event *host.TraceEvent) (*AddPeer, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for AddPeer")
	}

	peerID, ok := payload["PeerID"].(peer.ID)
	if !ok {
		return nil, fmt.Errorf("peerID is required for AddPeer")
	}

	protoc, ok := payload["Protocol"].(protocol.ID)
	if !ok {
		return nil, fmt.Errorf("protocol is required for AddPeer")
	}

	return &AddPeer{
		PeerId:   wrapperspb.String(peerID.String()),
		Protocol: wrapperspb.String(string(protoc)),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p RemovePeer
func TraceEventToRemovePeer(event *host.TraceEvent) (*RemovePeer, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for RemovePeer")
	}

	peerID, ok := payload["PeerID"].(peer.ID)
	if !ok {
		return nil, fmt.Errorf("peerID is required for RemovePeer")
	}

	return &RemovePeer{
		PeerId: wrapperspb.String(peerID.String()),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p Join
func TraceEventToJoin(event *host.TraceEvent) (*Join, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for Join")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for Join")
	}

	return &Join{
		Topic: wrapperspb.String(topic),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p Leave
func TraceEventToLeave(event *host.TraceEvent) (*Leave, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for Leave")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for Leave")
	}

	return &Leave{
		Topic: wrapperspb.String(topic),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p Graft.
func TraceEventToGraft(event *host.TraceEvent) (*Graft, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for Graft")
	}

	peerID, ok := payload["PeerID"].(peer.ID)
	if !ok {
		return nil, fmt.Errorf("peerID is required for Graft")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for Graft")
	}

	return &Graft{
		Topic:  wrapperspb.String(topic),
		PeerId: wrapperspb.String(peerID.String()),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p Prune.
func TraceEventToPrune(event *host.TraceEvent) (*Prune, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for Prune")
	}

	peerID, ok := payload["PeerID"].(peer.ID)
	if !ok {
		return nil, fmt.Errorf("peerID is required for Prune")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for Prune")
	}

	return &Prune{
		Topic:  wrapperspb.String(topic),
		PeerId: wrapperspb.String(peerID.String()),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p RecvRPC.
func TraceEventToRecvRPC(event *host.TraceEvent) (*RecvRPC, error) {
	payload, ok := event.Payload.(*host.RpcMeta)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for rpc")
	}

	r := &RecvRPC{
		PeerId: wrapperspb.String(payload.PeerID.String()),
		Meta: &RPCMeta{
			PeerId:        wrapperspb.String(payload.PeerID.String()),
			Messages:      convertRPCMessages(payload.Messages),
			Subscriptions: convertRPCSubscriptions(payload.Subscriptions),
			Control:       convertRPCControl(payload.Control),
		},
	}

	return r, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p SendRPC.
func TraceEventToSendRPC(event *host.TraceEvent) (*SendRPC, error) {
	payload, ok := event.Payload.(*host.RpcMeta)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for rpc")
	}

	r := &SendRPC{
		PeerId: wrapperspb.String(payload.PeerID.String()),
		Meta: &RPCMeta{
			PeerId:        wrapperspb.String(payload.PeerID.String()),
			Messages:      convertRPCMessages(payload.Messages),
			Subscriptions: convertRPCSubscriptions(payload.Subscriptions),
			Control:       convertRPCControl(payload.Control),
		},
	}

	return r, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p DropRPC.
func TraceEventToDropRPC(event *host.TraceEvent) (*DropRPC, error) {
	payload, ok := event.Payload.(*host.RpcMeta)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for rpc")
	}

	r := &DropRPC{
		PeerId: wrapperspb.String(payload.PeerID.String()),
		Meta: &RPCMeta{
			PeerId:        wrapperspb.String(payload.PeerID.String()),
			Messages:      convertRPCMessages(payload.Messages),
			Subscriptions: convertRPCSubscriptions(payload.Subscriptions),
			Control:       convertRPCControl(payload.Control),
		},
	}

	return r, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p PublishMessage.
func TraceEventToPublishMessage(event *host.TraceEvent) (*PublishMessage, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for PublishMessage")
	}

	msgID, ok := payload["MsgID"].(string)
	if !ok {
		return nil, fmt.Errorf("msgID is required for PublishMessage")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for PublishMessage")
	}

	return &PublishMessage{
		MsgId: wrapperspb.String(msgID),
		Topic: wrapperspb.String(topic),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p RejectMessage.
func TraceEventToRejectMessage(event *host.TraceEvent) (*RejectMessage, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for RejectMessage")
	}

	msgID, ok := payload["MsgID"].(string)
	if !ok {
		return nil, fmt.Errorf("msgID is required for RejectMessage")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for RejectMessage")
	}

	peerID, ok := payload["PeerID"].(peer.ID)
	if !ok {
		return nil, fmt.Errorf("peerID is required for RejectMessage")
	}

	reason, ok := payload["Reason"].(string)
	if !ok {
		return nil, fmt.Errorf("reason is required for RejectMessage")
	}

	local, ok := payload["Local"].(bool)
	if !ok {
		return nil, fmt.Errorf("local is required for RejectMessage")
	}

	msgSize, ok := payload["MsgSize"].(int)
	if !ok {
		return nil, fmt.Errorf("msgSize is required for RejectMessage")
	}

	seqHex, ok := payload["Seq"].(string)
	if !ok {
		return nil, fmt.Errorf("seq is required for RejectMessage")
	}

	// Parse hex sequence number.
	seqBytes, err := hex.DecodeString(seqHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Seq hex: %w", err)
	}

	var seqNumber uint64

	if len(seqBytes) > 0 {
		for _, b := range seqBytes {
			seqNumber = (seqNumber << 8) | uint64(b)
		}
	}

	return &RejectMessage{
		MsgId:     wrapperspb.String(msgID),
		PeerId:    wrapperspb.String(peerID.String()),
		Topic:     wrapperspb.String(topic),
		Reason:    wrapperspb.String(reason),
		Local:     wrapperspb.Bool(local),
		MsgSize:   wrapperspb.UInt32(uint32(msgSize)), //nolint:gosec // fine.
		SeqNumber: wrapperspb.UInt64(seqNumber),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p DuplicateMessage.
func TraceEventToDuplicateMessage(event *host.TraceEvent) (*DuplicateMessage, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for DuplicateMessage")
	}

	msgID, ok := payload["MsgID"].(string)
	if !ok {
		return nil, fmt.Errorf("msgID is required for DuplicateMessage")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for DuplicateMessage")
	}

	peerID, ok := payload["PeerID"].(peer.ID)
	if !ok {
		return nil, fmt.Errorf("peerID is required for DuplicateMessage")
	}

	local, ok := payload["Local"].(bool)
	if !ok {
		return nil, fmt.Errorf("local is required for DuplicateMessage")
	}

	msgSize, ok := payload["MsgSize"].(int)
	if !ok {
		return nil, fmt.Errorf("msgSize is required for DuplicateMessage")
	}

	seqHex, ok := payload["Seq"].(string)
	if !ok {
		return nil, fmt.Errorf("seq is required for DuplicateMessage")
	}

	// Parse hex sequence number
	seqBytes, err := hex.DecodeString(seqHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Seq hex: %w", err)
	}

	var seqNumber uint64

	if len(seqBytes) > 0 {
		for _, b := range seqBytes {
			seqNumber = (seqNumber << 8) | uint64(b)
		}
	}

	return &DuplicateMessage{
		MsgId:     wrapperspb.String(msgID),
		PeerId:    wrapperspb.String(peerID.String()),
		Topic:     wrapperspb.String(topic),
		Local:     wrapperspb.Bool(local),
		MsgSize:   wrapperspb.UInt32(uint32(msgSize)), //nolint:gosec // fine.
		SeqNumber: wrapperspb.UInt64(seqNumber),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p DeliverMessage.
func TraceEventToDeliverMessage(event *host.TraceEvent) (*DeliverMessage, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for DeliverMessage")
	}

	msgID, ok := payload["MsgID"].(string)
	if !ok {
		return nil, fmt.Errorf("msgID is required for DeliverMessage")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for DeliverMessage")
	}

	peerID, ok := payload["PeerID"].(peer.ID)
	if !ok {
		return nil, fmt.Errorf("peerID is required for DeliverMessage")
	}

	local, ok := payload["Local"].(bool)
	if !ok {
		return nil, fmt.Errorf("local is required for DeliverMessage")
	}

	msgSize, ok := payload["MsgSize"].(int)
	if !ok {
		return nil, fmt.Errorf("msgSize is required for DeliverMessage")
	}

	seqHex, ok := payload["Seq"].(string)
	if !ok {
		return nil, fmt.Errorf("seq is required for DeliverMessage")
	}

	// Parse hex sequence number
	seqBytes, err := hex.DecodeString(seqHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Seq hex: %w", err)
	}

	var seqNumber uint64

	if len(seqBytes) > 0 {
		for _, b := range seqBytes {
			seqNumber = (seqNumber << 8) | uint64(b)
		}
	}

	return &DeliverMessage{
		MsgId:     wrapperspb.String(msgID),
		PeerId:    wrapperspb.String(peerID.String()),
		Topic:     wrapperspb.String(topic),
		Local:     wrapperspb.Bool(local),
		MsgSize:   wrapperspb.UInt32(uint32(msgSize)), //nolint:gosec // fine.
		SeqNumber: wrapperspb.UInt64(seqNumber),
	}, nil
}

func convertRPCMessages(messages []host.RpcMetaMsg) []*MessageMeta {
	ourMessages := make([]*MessageMeta, len(messages))

	for i, msg := range messages {
		ourMessages[i] = &MessageMeta{
			MessageId: wrapperspb.String(msg.MsgID),
			TopicId:   wrapperspb.String(msg.Topic),
		}
	}

	return ourMessages
}

func convertRPCSubscriptions(subs []host.RpcMetaSub) []*SubMeta {
	ourSubs := make([]*SubMeta, len(subs))

	for i, sub := range subs {
		ourSubs[i] = &SubMeta{
			Subscribe: wrapperspb.Bool(sub.Subscribe),
			TopicId:   wrapperspb.String(sub.TopicID),
		}
	}

	return ourSubs
}

func convertRPCControl(ctrl *host.RpcMetaControl) *ControlMeta {
	if ctrl == nil {
		return nil
	}

	return &ControlMeta{
		Ihave:     convertControlIHaveMeta(ctrl.IHave),
		Iwant:     convertControlIWantMeta(ctrl.IWant),
		Graft:     convertControlGraftMeta(ctrl.Graft),
		Prune:     convertControlPruneMeta(ctrl.Prune),
		Idontwant: convertControlIDontWantMeta(ctrl.Idontwant),
	}
}

func convertControlIHaveMeta(ihave []host.RpcControlIHave) []*ControlIHaveMeta {
	converted := make([]*ControlIHaveMeta, len(ihave))

	for i, item := range ihave {
		converted[i] = &ControlIHaveMeta{
			TopicId:    wrapperspb.String(item.TopicID),
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlIWantMeta(iwant []host.RpcControlIWant) []*ControlIWantMeta {
	converted := make([]*ControlIWantMeta, len(iwant))

	for i, item := range iwant {
		converted[i] = &ControlIWantMeta{
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlIDontWantMeta(idontwant []host.RpcControlIdontWant) []*ControlIDontWantMeta {
	converted := make([]*ControlIDontWantMeta, len(idontwant))

	for i, item := range idontwant {
		converted[i] = &ControlIDontWantMeta{
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlGraftMeta(graft []host.RpcControlGraft) []*ControlGraftMeta {
	converted := make([]*ControlGraftMeta, len(graft))

	for i, item := range graft {
		converted[i] = &ControlGraftMeta{
			TopicId: wrapperspb.String(item.TopicID),
		}
	}

	return converted
}

func convertControlPruneMeta(prune []host.RpcControlPrune) []*ControlPruneMeta {
	converted := make([]*ControlPruneMeta, len(prune))

	for i, item := range prune {
		peerIds := make([]string, 0, len(item.PeerIDs))
		for _, peer := range item.PeerIDs {
			peerIds = append(peerIds, peer.String())
		}

		converted[i] = &ControlPruneMeta{
			TopicId: wrapperspb.String(item.TopicID),
			PeerIds: convertStringValues(peerIds),
		}
	}

	return converted
}

func convertStringValues(strings []string) []*wrapperspb.StringValue {
	converted := make([]*wrapperspb.StringValue, len(strings))

	for i, s := range strings {
		converted[i] = wrapperspb.String(s)
	}

	return converted
}

func TraceEventToConnected(event *host.TraceEvent) (*Connected, error) {
	payload, ok := event.Payload.(struct {
		RemotePeer   string
		RemoteMaddrs ma.Multiaddr
		AgentVersion string
		Direction    string
		Opened       time.Time
		Limited      bool
	})
	if !ok {
		return nil, fmt.Errorf("invalid payload type for Connected")
	}

	return &Connected{
		RemotePeer:   wrapperspb.String(payload.RemotePeer),
		RemoteMaddrs: wrapperspb.String(payload.RemoteMaddrs.String()),
		AgentVersion: wrapperspb.String(payload.AgentVersion),
		Direction:    wrapperspb.String(payload.Direction),
		Opened:       timestamppb.New(payload.Opened),
		Limited:      wrapperspb.Bool(payload.Limited),
		Transient:    wrapperspb.Bool(payload.Limited),
	}, nil
}

func TraceEventToDisconnected(event *host.TraceEvent) (*Disconnected, error) {
	payload, ok := event.Payload.(struct {
		RemotePeer   string
		RemoteMaddrs ma.Multiaddr
		AgentVersion string
		Direction    string
		Opened       time.Time
		Limited      bool
	})
	if !ok {
		return nil, fmt.Errorf("invalid payload type for Disconnected")
	}

	return &Disconnected{
		RemotePeer:   wrapperspb.String(payload.RemotePeer),
		RemoteMaddrs: wrapperspb.String(payload.RemoteMaddrs.String()),
		AgentVersion: wrapperspb.String(payload.AgentVersion),
		Direction:    wrapperspb.String(payload.Direction),
		Opened:       timestamppb.New(payload.Opened),
		Limited:      wrapperspb.Bool(payload.Limited),
		Transient:    wrapperspb.Bool(payload.Limited),
	}, nil
}

func TraceEventToHandleMetadata(event *host.TraceEvent) (*HandleMetadata, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for HandleMetadata")
	}

	metadata := &HandleMetadata{}

	if peerID, ok := payload["PeerID"].(peer.ID); ok {
		metadata.PeerId = wrapperspb.String(peerID.String())
	}

	if protocolID, ok := payload["ProtocolID"].(protocol.ID); ok {
		metadata.ProtocolId = wrapperspb.String(string(protocolID))
	}

	if latencyS, ok := payload["LatencyS"].(float64); ok {
		metadata.Latency = wrapperspb.Float(float32(latencyS))
	}

	metadata.Metadata = &Metadata{}

	if seqNumber, ok := payload["SeqNumber"].(uint64); ok {
		metadata.Metadata.SeqNumber = wrapperspb.UInt64(seqNumber)
	}

	if attnets, ok := payload["Attnets"].(string); ok {
		metadata.Metadata.Attnets = wrapperspb.String(attnets)
	}

	if syncnets, ok := payload["Syncnets"].(string); ok {
		metadata.Metadata.Syncnets = wrapperspb.String(syncnets)
	}

	if errorStr, ok := payload["Error"].(string); ok {
		metadata.Error = wrapperspb.String(errorStr)
	}

	return metadata, nil
}

func TraceEventToHandleStatus(event *host.TraceEvent) (*HandleStatus, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for HandleStatus")
	}

	status := &HandleStatus{}

	if peerID, ok := payload["PeerID"].(peer.ID); ok {
		status.PeerId = wrapperspb.String(peerID.String())
	}

	if protocolID, ok := payload["ProtocolID"].(protocol.ID); ok {
		status.ProtocolId = wrapperspb.String(string(protocolID))
	}

	if latencyS, ok := payload["LatencyS"].(float64); ok {
		status.Latency = wrapperspb.Float(float32(latencyS))
	}

	if requestData, ok := payload["Request"].(map[string]any); ok {
		request, err := parseStatus(requestData)
		if err != nil {
			return nil, err
		}

		status.Request = &Status{
			ForkDigest:     wrapperspb.String(request.ForkDigest),
			FinalizedRoot:  wrapperspb.String(request.FinalizedRoot),
			FinalizedEpoch: wrapperspb.UInt64(request.FinalizedEpoch),
			HeadRoot:       wrapperspb.String(request.HeadRoot),
			HeadSlot:       wrapperspb.UInt64(request.HeadSlot),
		}
	}

	if responseData, ok := payload["Response"].(map[string]any); ok {
		response, err := parseStatus(responseData)
		if err != nil {
			return nil, err
		}

		status.Response = &Status{
			ForkDigest:     wrapperspb.String(response.ForkDigest),
			FinalizedRoot:  wrapperspb.String(response.FinalizedRoot),
			FinalizedEpoch: wrapperspb.UInt64(response.FinalizedEpoch),
			HeadRoot:       wrapperspb.String(response.HeadRoot),
			HeadSlot:       wrapperspb.UInt64(response.HeadSlot),
		}
	}

	if errorStr, ok := payload["Error"].(string); ok {
		status.Error = wrapperspb.String(errorStr)
	}

	return status, nil
}

type statusFields struct {
	ForkDigest     string
	FinalizedRoot  string
	FinalizedEpoch uint64
	HeadRoot       string
	HeadSlot       uint64
}

func parseStatus(data map[string]any) (*statusFields, error) {
	status := &statusFields{}

	if forkDigest, ok := data["ForkDigest"].(string); ok {
		status.ForkDigest = forkDigest
	}

	if finalizedRoot, ok := data["FinalizedRoot"].(string); ok {
		status.FinalizedRoot = finalizedRoot
	}

	if finalizedEpoch, ok := data["FinalizedEpoch"].(primitives.Epoch); ok {
		status.FinalizedEpoch = uint64(finalizedEpoch)
	}

	if headRoot, ok := data["HeadRoot"].(string); ok {
		status.HeadRoot = headRoot
	}

	if headSlot, ok := data["HeadSlot"].(primitives.Slot); ok {
		status.HeadSlot = uint64(headSlot)
	}

	return status, nil
}
