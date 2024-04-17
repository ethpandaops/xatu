package libp2p

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/probe-lab/hermes/host"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func EventTypeFromHermesEventType(e host.EventType) EventType {
	if val, ok := EventType_value[string(e)]; ok {
		return EventType(val)
	}

	return EventType(0) // Return an Unknown EventType if not found
}

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

// Helper function to convert a Hermes TraceEvent to a libp2p RecvRPC
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

// Helper function to convert a Hermes TraceEvent to a libp2p SendRPC
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

func convertRPCMessages(messages []host.RpcMetaMsg) []*MessageMeta {
	ourMessages := make([]*MessageMeta, len(messages))

	for i, msg := range messages {
		ourMessages[i] = convertRPCMetaMessage(msg)
	}

	return ourMessages
}

func convertRPCMetaMessage(msg host.RpcMetaMsg) *MessageMeta {
	return &MessageMeta{
		MessageId: wrapperspb.String(msg.MsgID),
		Topic:     wrapperspb.String(msg.Topic),
	}
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
		Ihave: convertControlIHaveMeta(ctrl.IHave),
		Iwant: convertControlIWantMeta(ctrl.IWant),
		Graft: convertControlGraftMeta(ctrl.Graft),
		Prune: convertControlPruneMeta(ctrl.Prune),
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
		peerIds := make([]string, len(prune))
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
		Transient    bool
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
		Transient:    wrapperspb.Bool(payload.Transient),
	}, nil
}

func TraceEventToDisconnected(event *host.TraceEvent) (*Disconnected, error) {
	payload, ok := event.Payload.(struct {
		RemotePeer   string
		RemoteMaddrs ma.Multiaddr
		AgentVersion string
		Direction    string
		Opened       time.Time
		Transient    bool
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
		Transient:    wrapperspb.Bool(payload.Transient),
	}, nil
}
