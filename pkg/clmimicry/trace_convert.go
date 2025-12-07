package clmimicry

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
)

// Helper function to convert a Hermes TraceEvent to a libp2p AddPeer
func TraceEventToAddPeer(event *TraceEvent) (*libp2p.AddPeer, error) {
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

	return &libp2p.AddPeer{
		PeerId:   wrapperspb.String(peerID.String()),
		Protocol: wrapperspb.String(string(protoc)),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p RemovePeer
func TraceEventToRemovePeer(event *TraceEvent) (*libp2p.RemovePeer, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for RemovePeer")
	}

	peerID, ok := payload["PeerID"].(peer.ID)
	if !ok {
		return nil, fmt.Errorf("peerID is required for RemovePeer")
	}

	return &libp2p.RemovePeer{
		PeerId: wrapperspb.String(peerID.String()),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p Join
func TraceEventToJoin(event *TraceEvent) (*libp2p.Join, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for Join")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for Join")
	}

	return &libp2p.Join{
		Topic: wrapperspb.String(topic),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p Leave
func TraceEventToLeave(event *TraceEvent) (*libp2p.Leave, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for Leave")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("topic is required for Leave")
	}

	return &libp2p.Leave{
		Topic: wrapperspb.String(topic),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p Graft.
func TraceEventToGraft(event *TraceEvent) (*libp2p.Graft, error) {
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

	return &libp2p.Graft{
		Topic:  wrapperspb.String(topic),
		PeerId: wrapperspb.String(peerID.String()),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p Prune.
func TraceEventToPrune(event *TraceEvent) (*libp2p.Prune, error) {
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

	return &libp2p.Prune{
		Topic:  wrapperspb.String(topic),
		PeerId: wrapperspb.String(peerID.String()),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p RecvRPC.
func TraceEventToRecvRPC(event *TraceEvent) (*libp2p.RecvRPC, error) {
	payload, ok := event.Payload.(*RpcMeta)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for rpc")
	}

	r := &libp2p.RecvRPC{
		PeerId: wrapperspb.String(payload.PeerID.String()),
		Meta: &libp2p.RPCMeta{
			PeerId:        wrapperspb.String(payload.PeerID.String()),
			Messages:      convertRPCMessages(payload.Messages),
			Subscriptions: convertRPCSubscriptions(payload.Subscriptions),
			Control:       convertRPCControl(payload.Control),
		},
	}

	return r, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p SendRPC.
func TraceEventToSendRPC(event *TraceEvent) (*libp2p.SendRPC, error) {
	payload, ok := event.Payload.(*RpcMeta)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for rpc")
	}

	r := &libp2p.SendRPC{
		PeerId: wrapperspb.String(payload.PeerID.String()),
		Meta: &libp2p.RPCMeta{
			PeerId:        wrapperspb.String(payload.PeerID.String()),
			Messages:      convertRPCMessages(payload.Messages),
			Subscriptions: convertRPCSubscriptions(payload.Subscriptions),
			Control:       convertRPCControl(payload.Control),
		},
	}

	return r, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p DropRPC.
func TraceEventToDropRPC(event *TraceEvent) (*libp2p.DropRPC, error) {
	payload, ok := event.Payload.(*RpcMeta)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for rpc")
	}

	r := &libp2p.DropRPC{
		PeerId: wrapperspb.String(payload.PeerID.String()),
		Meta: &libp2p.RPCMeta{
			PeerId:        wrapperspb.String(payload.PeerID.String()),
			Messages:      convertRPCMessages(payload.Messages),
			Subscriptions: convertRPCSubscriptions(payload.Subscriptions),
			Control:       convertRPCControl(payload.Control),
		},
	}

	return r, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p PublishMessage.
func TraceEventToPublishMessage(event *TraceEvent) (*libp2p.PublishMessage, error) {
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

	return &libp2p.PublishMessage{
		MsgId: wrapperspb.String(msgID),
		Topic: wrapperspb.String(topic),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p RejectMessage.
func TraceEventToRejectMessage(event *TraceEvent) (*libp2p.RejectMessage, error) {
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

	return &libp2p.RejectMessage{
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
func TraceEventToDuplicateMessage(event *TraceEvent) (*libp2p.DuplicateMessage, error) {
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

	return &libp2p.DuplicateMessage{
		MsgId:     wrapperspb.String(msgID),
		PeerId:    wrapperspb.String(peerID.String()),
		Topic:     wrapperspb.String(topic),
		Local:     wrapperspb.Bool(local),
		MsgSize:   wrapperspb.UInt32(uint32(msgSize)), //nolint:gosec // fine.
		SeqNumber: wrapperspb.UInt64(seqNumber),
	}, nil
}

// Helper function to convert a Hermes TraceEvent to a libp2p DeliverMessage.
func TraceEventToDeliverMessage(event *TraceEvent) (*libp2p.DeliverMessage, error) {
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

	return &libp2p.DeliverMessage{
		MsgId:     wrapperspb.String(msgID),
		PeerId:    wrapperspb.String(peerID.String()),
		Topic:     wrapperspb.String(topic),
		Local:     wrapperspb.Bool(local),
		MsgSize:   wrapperspb.UInt32(uint32(msgSize)), //nolint:gosec // fine.
		SeqNumber: wrapperspb.UInt64(seqNumber),
	}, nil
}

func convertRPCMessages(messages []RpcMetaMsg) []*libp2p.MessageMeta {
	ourMessages := make([]*libp2p.MessageMeta, len(messages))

	for i, msg := range messages {
		ourMessages[i] = &libp2p.MessageMeta{
			MessageId: wrapperspb.String(msg.MsgID),
			TopicId:   wrapperspb.String(msg.Topic),
		}
	}

	return ourMessages
}

func convertRPCSubscriptions(subs []RpcMetaSub) []*libp2p.SubMeta {
	ourSubs := make([]*libp2p.SubMeta, len(subs))

	for i, sub := range subs {
		ourSubs[i] = &libp2p.SubMeta{
			Subscribe: wrapperspb.Bool(sub.Subscribe),
			TopicId:   wrapperspb.String(sub.TopicID),
		}
	}

	return ourSubs
}

func convertRPCControl(ctrl *RpcMetaControl) *libp2p.ControlMeta {
	if ctrl == nil {
		return nil
	}

	return &libp2p.ControlMeta{
		Ihave:     convertControlIHaveMeta(ctrl.IHave),
		Iwant:     convertControlIWantMeta(ctrl.IWant),
		Graft:     convertControlGraftMeta(ctrl.Graft),
		Prune:     convertControlPruneMeta(ctrl.Prune),
		Idontwant: convertControlIDontWantMeta(ctrl.Idontwant),
	}
}

func convertControlIHaveMeta(ihave []RpcControlIHave) []*libp2p.ControlIHaveMeta {
	converted := make([]*libp2p.ControlIHaveMeta, len(ihave))

	for i, item := range ihave {
		converted[i] = &libp2p.ControlIHaveMeta{
			TopicId:    wrapperspb.String(item.TopicID),
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlIWantMeta(iwant []RpcControlIWant) []*libp2p.ControlIWantMeta {
	converted := make([]*libp2p.ControlIWantMeta, len(iwant))

	for i, item := range iwant {
		converted[i] = &libp2p.ControlIWantMeta{
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlIDontWantMeta(idontwant []RpcControlIdontWant) []*libp2p.ControlIDontWantMeta {
	converted := make([]*libp2p.ControlIDontWantMeta, len(idontwant))

	for i, item := range idontwant {
		converted[i] = &libp2p.ControlIDontWantMeta{
			MessageIds: convertStringValues(item.MsgIDs),
		}
	}

	return converted
}

func convertControlGraftMeta(graft []RpcControlGraft) []*libp2p.ControlGraftMeta {
	converted := make([]*libp2p.ControlGraftMeta, len(graft))

	for i, item := range graft {
		converted[i] = &libp2p.ControlGraftMeta{
			TopicId: wrapperspb.String(item.TopicID),
		}
	}

	return converted
}

func convertControlPruneMeta(prune []RpcControlPrune) []*libp2p.ControlPruneMeta {
	converted := make([]*libp2p.ControlPruneMeta, len(prune))

	for i, item := range prune {
		peerIds := make([]string, 0, len(item.PeerIDs))
		for _, peer := range item.PeerIDs {
			peerIds = append(peerIds, peer.String())
		}

		converted[i] = &libp2p.ControlPruneMeta{
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

func TraceEventToConnected(event *TraceEvent) (*libp2p.Connected, error) {
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

	return &libp2p.Connected{
		RemotePeer:   wrapperspb.String(payload.RemotePeer),
		RemoteMaddrs: wrapperspb.String(payload.RemoteMaddrs.String()),
		AgentVersion: wrapperspb.String(payload.AgentVersion),
		Direction:    wrapperspb.String(payload.Direction),
		Opened:       timestamppb.New(payload.Opened),
		Limited:      wrapperspb.Bool(payload.Limited),
		Transient:    wrapperspb.Bool(payload.Limited),
	}, nil
}

func TraceEventToDisconnected(event *TraceEvent) (*libp2p.Disconnected, error) {
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

	return &libp2p.Disconnected{
		RemotePeer:   wrapperspb.String(payload.RemotePeer),
		RemoteMaddrs: wrapperspb.String(payload.RemoteMaddrs.String()),
		AgentVersion: wrapperspb.String(payload.AgentVersion),
		Direction:    wrapperspb.String(payload.Direction),
		Opened:       timestamppb.New(payload.Opened),
		Limited:      wrapperspb.Bool(payload.Limited),
		Transient:    wrapperspb.Bool(payload.Limited),
	}, nil
}

func TraceEventToHandleMetadata(event *TraceEvent) (*libp2p.HandleMetadata, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for HandleMetadata")
	}

	metadata := &libp2p.HandleMetadata{}

	if peerID, ok := payload["PeerID"].(peer.ID); ok {
		metadata.PeerId = wrapperspb.String(peerID.String())
	}

	if protocolID, ok := payload["ProtocolID"].(protocol.ID); ok {
		metadata.ProtocolId = wrapperspb.String(string(protocolID))
	}

	if latencyS, ok := payload["LatencyS"].(float64); ok {
		metadata.Latency = wrapperspb.Float(float32(latencyS))
	}

	metadata.Metadata = &libp2p.Metadata{}

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

	if direction, ok := payload["Direction"].(string); ok {
		metadata.Direction = wrapperspb.String(direction)
	}

	if custodyGroupCount, ok := payload["CustodyGroupCount"].(uint64); ok {
		metadata.Metadata.CustodyGroupCount = wrapperspb.UInt64(custodyGroupCount)
	}

	return metadata, nil
}

func TraceEventToHandleStatus(event *TraceEvent) (*libp2p.HandleStatus, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for HandleStatus")
	}

	status := &libp2p.HandleStatus{}

	if peerID, ok := payload["PeerID"].(peer.ID); ok {
		status.PeerId = wrapperspb.String(peerID.String())
	}

	if protocolID, ok := payload["ProtocolID"].(protocol.ID); ok {
		status.ProtocolId = wrapperspb.String(string(protocolID))
	}

	if latencyS, ok := payload["LatencyS"].(float64); ok {
		status.Latency = wrapperspb.Float(float32(latencyS))
	}

	if direction, ok := payload["Direction"].(string); ok {
		status.Direction = wrapperspb.String(direction)
	}

	if requestData, ok := payload["Request"].(map[string]any); ok {
		request, err := parseStatus(requestData)
		if err != nil {
			return nil, err
		}

		status.Request = &libp2p.Status{
			ForkDigest:            wrapperspb.String(request.ForkDigest),
			FinalizedRoot:         wrapperspb.String(request.FinalizedRoot),
			FinalizedEpoch:        wrapperspb.UInt64(request.FinalizedEpoch),
			HeadRoot:              wrapperspb.String(request.HeadRoot),
			HeadSlot:              wrapperspb.UInt64(request.HeadSlot),
			EarliestAvailableSlot: wrapperspb.UInt64(request.EarliestAvailableSlot),
		}
	}

	if responseData, ok := payload["Response"].(map[string]any); ok {
		response, err := parseStatus(responseData)
		if err != nil {
			return nil, err
		}

		status.Response = &libp2p.Status{
			ForkDigest:            wrapperspb.String(response.ForkDigest),
			FinalizedRoot:         wrapperspb.String(response.FinalizedRoot),
			FinalizedEpoch:        wrapperspb.UInt64(response.FinalizedEpoch),
			HeadRoot:              wrapperspb.String(response.HeadRoot),
			HeadSlot:              wrapperspb.UInt64(response.HeadSlot),
			EarliestAvailableSlot: wrapperspb.UInt64(response.EarliestAvailableSlot),
		}
	}

	if errorStr, ok := payload["Error"].(string); ok {
		status.Error = wrapperspb.String(errorStr)
	}

	return status, nil
}

type statusFields struct {
	ForkDigest            string
	FinalizedRoot         string
	FinalizedEpoch        uint64
	HeadRoot              string
	HeadSlot              uint64
	EarliestAvailableSlot uint64
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

	if earliestAvailableSlot, ok := data["EarliestAvailableSlot"].(primitives.Slot); ok {
		status.EarliestAvailableSlot = uint64(earliestAvailableSlot)
	}

	return status, nil
}

// TraceEventToSyntheticHeartbeat converts a Hermes TraceEvent to a SyntheticHeartbeat protobuf message
func TraceEventToSyntheticHeartbeat(event *TraceEvent) (*libp2p.SyntheticHeartbeat, error) {
	// The payload structure for heartbeat events from Hermes
	payload, ok := event.Payload.(struct {
		RemotePeer      string
		RemoteMaddrs    ma.Multiaddr
		LatencyMs       int64
		AgentVersion    string
		Direction       uint32
		Protocols       []string
		ConnectionAgeNs int64
	})
	if !ok {
		return nil, fmt.Errorf("invalid payload type for SyntheticHeartbeat")
	}

	return &libp2p.SyntheticHeartbeat{
		Timestamp:       timestamppb.New(event.Timestamp),
		RemotePeer:      wrapperspb.String(payload.RemotePeer),
		RemoteMaddrs:    wrapperspb.String(payload.RemoteMaddrs.String()),
		LatencyMs:       wrapperspb.Int64(payload.LatencyMs),
		AgentVersion:    wrapperspb.String(payload.AgentVersion),
		Direction:       wrapperspb.UInt32(payload.Direction),
		Protocols:       payload.Protocols,
		ConnectionAgeNs: wrapperspb.Int64(payload.ConnectionAgeNs),
	}, nil
}

// TraceEventToCustodyProbe converts a Hermes TraceEvent to a DataColumnCustodyProbe protobuf message
func TraceEventToCustodyProbe(event *TraceEvent) (*libp2p.DataColumnCustodyProbe, error) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for CustodyProbe")
	}

	probe := &libp2p.DataColumnCustodyProbe{}

	if jobStartTimestamp, ok := payload["JobStartTimestamp"].(time.Time); ok {
		probe.JobStartTimestamp = timestamppb.New(jobStartTimestamp)
	}

	if peerID, ok := payload["PeerID"].(string); ok {
		probe.PeerId = wrapperspb.String(peerID)
	}

	if slot, ok := payload["Slot"].(uint64); ok {
		//nolint:gosec // conversion fine.
		probe.Slot = wrapperspb.UInt32(uint32(slot))
	}

	if epoch, ok := payload["Epoch"].(uint64); ok {
		//nolint:gosec // conversion fine.
		probe.Epoch = wrapperspb.UInt32(uint32(epoch))
	}

	if columnIndex, ok := payload["ColumnIndex"].(uint64); ok {
		//nolint:gosec // conversion fine.
		probe.ColumnIndex = wrapperspb.UInt32(uint32(columnIndex))
	}

	if result, ok := payload["Result"].(string); ok {
		probe.Result = wrapperspb.String(result)
	}

	if responseTimeMs, ok := payload["DurationMs"].(int64); ok {
		probe.ResponseTimeMs = wrapperspb.Int64(responseTimeMs)
	}

	if errorStr, ok := payload["Error"].(string); ok {
		probe.Error = wrapperspb.String(errorStr)
	}

	if beaconBlockRoot, ok := payload["BlockHash"].(string); ok {
		probe.BeaconBlockRoot = wrapperspb.String(beaconBlockRoot)
	}

	if columnRowsCount, ok := payload["ColumnSize"].(int); ok {
		//nolint:gosec // conversion fine.
		probe.ColumnRowsCount = wrapperspb.UInt32(uint32(columnRowsCount))
	}

	return probe, nil
}
