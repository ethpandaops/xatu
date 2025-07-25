// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: pkg/proto/noderecord/consensus.proto

package noderecord

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Consensus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Enr                         *wrapperspb.StringValue `protobuf:"bytes,1,opt,name=enr,proto3" json:"enr,omitempty"`
	NodeId                      *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=node_id,proto3" json:"node_id,omitempty"`
	PeerId                      *wrapperspb.StringValue `protobuf:"bytes,3,opt,name=peer_id,proto3" json:"peer_id,omitempty"`
	Timestamp                   *wrapperspb.Int64Value  `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Name                        *wrapperspb.StringValue `protobuf:"bytes,5,opt,name=name,proto3" json:"name,omitempty"`
	ForkDigest                  *wrapperspb.StringValue `protobuf:"bytes,6,opt,name=fork_digest,proto3" json:"fork_digest,omitempty"`
	NextForkDigest              *wrapperspb.StringValue `protobuf:"bytes,7,opt,name=next_fork_digest,proto3" json:"next_fork_digest,omitempty"`
	FinalizedRoot               *wrapperspb.StringValue `protobuf:"bytes,8,opt,name=finalized_root,proto3" json:"finalized_root,omitempty"`
	FinalizedEpoch              *wrapperspb.UInt64Value `protobuf:"bytes,9,opt,name=finalized_epoch,proto3" json:"finalized_epoch,omitempty"`
	FinalizedEpochStartDateTime *timestamppb.Timestamp  `protobuf:"bytes,10,opt,name=finalized_epoch_start_date_time,proto3" json:"finalized_epoch_start_date_time,omitempty"`
	HeadRoot                    *wrapperspb.StringValue `protobuf:"bytes,11,opt,name=head_root,proto3" json:"head_root,omitempty"`
	HeadSlot                    *wrapperspb.UInt64Value `protobuf:"bytes,12,opt,name=head_slot,proto3" json:"head_slot,omitempty"`
	HeadSlotStartDateTime       *timestamppb.Timestamp  `protobuf:"bytes,13,opt,name=head_slot_start_date_time,proto3" json:"head_slot_start_date_time,omitempty"`
	Cgc                         *wrapperspb.StringValue `protobuf:"bytes,14,opt,name=cgc,proto3" json:"cgc,omitempty"`
	Ip                          *wrapperspb.StringValue `protobuf:"bytes,15,opt,name=ip,proto3" json:"ip,omitempty"`
	Tcp                         *wrapperspb.UInt32Value `protobuf:"bytes,16,opt,name=tcp,proto3" json:"tcp,omitempty"`
	Udp                         *wrapperspb.UInt32Value `protobuf:"bytes,17,opt,name=udp,proto3" json:"udp,omitempty"`
	Quic                        *wrapperspb.UInt32Value `protobuf:"bytes,18,opt,name=quic,proto3" json:"quic,omitempty"`
	HasIpv6                     *wrapperspb.BoolValue   `protobuf:"bytes,19,opt,name=has_ipv6,proto3" json:"has_ipv6,omitempty"`
}

func (x *Consensus) Reset() {
	*x = Consensus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_noderecord_consensus_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Consensus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Consensus) ProtoMessage() {}

func (x *Consensus) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_noderecord_consensus_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Consensus.ProtoReflect.Descriptor instead.
func (*Consensus) Descriptor() ([]byte, []int) {
	return file_pkg_proto_noderecord_consensus_proto_rawDescGZIP(), []int{0}
}

func (x *Consensus) GetEnr() *wrapperspb.StringValue {
	if x != nil {
		return x.Enr
	}
	return nil
}

func (x *Consensus) GetNodeId() *wrapperspb.StringValue {
	if x != nil {
		return x.NodeId
	}
	return nil
}

func (x *Consensus) GetPeerId() *wrapperspb.StringValue {
	if x != nil {
		return x.PeerId
	}
	return nil
}

func (x *Consensus) GetTimestamp() *wrapperspb.Int64Value {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Consensus) GetName() *wrapperspb.StringValue {
	if x != nil {
		return x.Name
	}
	return nil
}

func (x *Consensus) GetForkDigest() *wrapperspb.StringValue {
	if x != nil {
		return x.ForkDigest
	}
	return nil
}

func (x *Consensus) GetNextForkDigest() *wrapperspb.StringValue {
	if x != nil {
		return x.NextForkDigest
	}
	return nil
}

func (x *Consensus) GetFinalizedRoot() *wrapperspb.StringValue {
	if x != nil {
		return x.FinalizedRoot
	}
	return nil
}

func (x *Consensus) GetFinalizedEpoch() *wrapperspb.UInt64Value {
	if x != nil {
		return x.FinalizedEpoch
	}
	return nil
}

func (x *Consensus) GetFinalizedEpochStartDateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.FinalizedEpochStartDateTime
	}
	return nil
}

func (x *Consensus) GetHeadRoot() *wrapperspb.StringValue {
	if x != nil {
		return x.HeadRoot
	}
	return nil
}

func (x *Consensus) GetHeadSlot() *wrapperspb.UInt64Value {
	if x != nil {
		return x.HeadSlot
	}
	return nil
}

func (x *Consensus) GetHeadSlotStartDateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.HeadSlotStartDateTime
	}
	return nil
}

func (x *Consensus) GetCgc() *wrapperspb.StringValue {
	if x != nil {
		return x.Cgc
	}
	return nil
}

func (x *Consensus) GetIp() *wrapperspb.StringValue {
	if x != nil {
		return x.Ip
	}
	return nil
}

func (x *Consensus) GetTcp() *wrapperspb.UInt32Value {
	if x != nil {
		return x.Tcp
	}
	return nil
}

func (x *Consensus) GetUdp() *wrapperspb.UInt32Value {
	if x != nil {
		return x.Udp
	}
	return nil
}

func (x *Consensus) GetQuic() *wrapperspb.UInt32Value {
	if x != nil {
		return x.Quic
	}
	return nil
}

func (x *Consensus) GetHasIpv6() *wrapperspb.BoolValue {
	if x != nil {
		return x.HasIpv6
	}
	return nil
}

var File_pkg_proto_noderecord_consensus_proto protoreflect.FileDescriptor

var file_pkg_proto_noderecord_consensus_proto_rawDesc = []byte{
	0x0a, 0x24, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6e, 0x6f, 0x64, 0x65,
	0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2f, 0x63, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x6e, 0x6f, 0x64,
	0x65, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x90, 0x09, 0x0a, 0x09, 0x43, 0x6f, 0x6e,
	0x73, 0x65, 0x6e, 0x73, 0x75, 0x73, 0x12, 0x2e, 0x0a, 0x03, 0x65, 0x6e, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x52, 0x03, 0x65, 0x6e, 0x72, 0x12, 0x36, 0x0a, 0x07, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x69,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x07, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x69, 0x64, 0x12, 0x36,
	0x0a, 0x07, 0x70, 0x65, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x07, 0x70,
	0x65, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x12, 0x39, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x49, 0x6e, 0x74, 0x36,
	0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x12, 0x30, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x3e, 0x0a, 0x0b, 0x66, 0x6f, 0x72, 0x6b, 0x5f, 0x64, 0x69, 0x67, 0x65,
	0x73, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e,
	0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0b, 0x66, 0x6f, 0x72, 0x6b, 0x5f, 0x64, 0x69, 0x67,
	0x65, 0x73, 0x74, 0x12, 0x48, 0x0a, 0x10, 0x6e, 0x65, 0x78, 0x74, 0x5f, 0x66, 0x6f, 0x72, 0x6b,
	0x5f, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x10, 0x6e, 0x65, 0x78,
	0x74, 0x5f, 0x66, 0x6f, 0x72, 0x6b, 0x5f, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x12, 0x44, 0x0a,
	0x0e, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x18,
	0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x52, 0x0e, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x72,
	0x6f, 0x6f, 0x74, 0x12, 0x46, 0x0a, 0x0f, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64,
	0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55,
	0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0f, 0x66, 0x69, 0x6e, 0x61,
	0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x64, 0x0a, 0x1f, 0x66,
	0x69, 0x6e, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x5f, 0x73,
	0x74, 0x61, 0x72, 0x74, 0x5f, 0x64, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x0a,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x1f, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x5f, 0x65, 0x70, 0x6f, 0x63,
	0x68, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74, 0x5f, 0x64, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d,
	0x65, 0x12, 0x3a, 0x0a, 0x09, 0x68, 0x65, 0x61, 0x64, 0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x0b,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x52, 0x09, 0x68, 0x65, 0x61, 0x64, 0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x12, 0x3a, 0x0a,
	0x09, 0x68, 0x65, 0x61, 0x64, 0x5f, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x09,
	0x68, 0x65, 0x61, 0x64, 0x5f, 0x73, 0x6c, 0x6f, 0x74, 0x12, 0x58, 0x0a, 0x19, 0x68, 0x65, 0x61,
	0x64, 0x5f, 0x73, 0x6c, 0x6f, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74, 0x5f, 0x64, 0x61, 0x74,
	0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x19, 0x68, 0x65, 0x61, 0x64, 0x5f, 0x73,
	0x6c, 0x6f, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74, 0x5f, 0x64, 0x61, 0x74, 0x65, 0x5f, 0x74,
	0x69, 0x6d, 0x65, 0x12, 0x2e, 0x0a, 0x03, 0x63, 0x67, 0x63, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x03,
	0x63, 0x67, 0x63, 0x12, 0x2c, 0x0a, 0x02, 0x69, 0x70, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x02, 0x69,
	0x70, 0x12, 0x2e, 0x0a, 0x03, 0x74, 0x63, 0x70, 0x18, 0x10, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x55, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x03, 0x74, 0x63,
	0x70, 0x12, 0x2e, 0x0a, 0x03, 0x75, 0x64, 0x70, 0x18, 0x11, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x55, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x03, 0x75, 0x64,
	0x70, 0x12, 0x30, 0x0a, 0x04, 0x71, 0x75, 0x69, 0x63, 0x18, 0x12, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x04, 0x71,
	0x75, 0x69, 0x63, 0x12, 0x36, 0x0a, 0x08, 0x68, 0x61, 0x73, 0x5f, 0x69, 0x70, 0x76, 0x36, 0x18,
	0x13, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x52, 0x08, 0x68, 0x61, 0x73, 0x5f, 0x69, 0x70, 0x76, 0x36, 0x42, 0x32, 0x5a, 0x30, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x74, 0x68, 0x70, 0x61, 0x6e,
	0x64, 0x61, 0x6f, 0x70, 0x73, 0x2f, 0x78, 0x61, 0x74, 0x75, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_proto_noderecord_consensus_proto_rawDescOnce sync.Once
	file_pkg_proto_noderecord_consensus_proto_rawDescData = file_pkg_proto_noderecord_consensus_proto_rawDesc
)

func file_pkg_proto_noderecord_consensus_proto_rawDescGZIP() []byte {
	file_pkg_proto_noderecord_consensus_proto_rawDescOnce.Do(func() {
		file_pkg_proto_noderecord_consensus_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_proto_noderecord_consensus_proto_rawDescData)
	})
	return file_pkg_proto_noderecord_consensus_proto_rawDescData
}

var file_pkg_proto_noderecord_consensus_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_pkg_proto_noderecord_consensus_proto_goTypes = []any{
	(*Consensus)(nil),              // 0: xatu.noderecord.Consensus
	(*wrapperspb.StringValue)(nil), // 1: google.protobuf.StringValue
	(*wrapperspb.Int64Value)(nil),  // 2: google.protobuf.Int64Value
	(*wrapperspb.UInt64Value)(nil), // 3: google.protobuf.UInt64Value
	(*timestamppb.Timestamp)(nil),  // 4: google.protobuf.Timestamp
	(*wrapperspb.UInt32Value)(nil), // 5: google.protobuf.UInt32Value
	(*wrapperspb.BoolValue)(nil),   // 6: google.protobuf.BoolValue
}
var file_pkg_proto_noderecord_consensus_proto_depIdxs = []int32{
	1,  // 0: xatu.noderecord.Consensus.enr:type_name -> google.protobuf.StringValue
	1,  // 1: xatu.noderecord.Consensus.node_id:type_name -> google.protobuf.StringValue
	1,  // 2: xatu.noderecord.Consensus.peer_id:type_name -> google.protobuf.StringValue
	2,  // 3: xatu.noderecord.Consensus.timestamp:type_name -> google.protobuf.Int64Value
	1,  // 4: xatu.noderecord.Consensus.name:type_name -> google.protobuf.StringValue
	1,  // 5: xatu.noderecord.Consensus.fork_digest:type_name -> google.protobuf.StringValue
	1,  // 6: xatu.noderecord.Consensus.next_fork_digest:type_name -> google.protobuf.StringValue
	1,  // 7: xatu.noderecord.Consensus.finalized_root:type_name -> google.protobuf.StringValue
	3,  // 8: xatu.noderecord.Consensus.finalized_epoch:type_name -> google.protobuf.UInt64Value
	4,  // 9: xatu.noderecord.Consensus.finalized_epoch_start_date_time:type_name -> google.protobuf.Timestamp
	1,  // 10: xatu.noderecord.Consensus.head_root:type_name -> google.protobuf.StringValue
	3,  // 11: xatu.noderecord.Consensus.head_slot:type_name -> google.protobuf.UInt64Value
	4,  // 12: xatu.noderecord.Consensus.head_slot_start_date_time:type_name -> google.protobuf.Timestamp
	1,  // 13: xatu.noderecord.Consensus.cgc:type_name -> google.protobuf.StringValue
	1,  // 14: xatu.noderecord.Consensus.ip:type_name -> google.protobuf.StringValue
	5,  // 15: xatu.noderecord.Consensus.tcp:type_name -> google.protobuf.UInt32Value
	5,  // 16: xatu.noderecord.Consensus.udp:type_name -> google.protobuf.UInt32Value
	5,  // 17: xatu.noderecord.Consensus.quic:type_name -> google.protobuf.UInt32Value
	6,  // 18: xatu.noderecord.Consensus.has_ipv6:type_name -> google.protobuf.BoolValue
	19, // [19:19] is the sub-list for method output_type
	19, // [19:19] is the sub-list for method input_type
	19, // [19:19] is the sub-list for extension type_name
	19, // [19:19] is the sub-list for extension extendee
	0,  // [0:19] is the sub-list for field type_name
}

func init() { file_pkg_proto_noderecord_consensus_proto_init() }
func file_pkg_proto_noderecord_consensus_proto_init() {
	if File_pkg_proto_noderecord_consensus_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_proto_noderecord_consensus_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*Consensus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pkg_proto_noderecord_consensus_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_proto_noderecord_consensus_proto_goTypes,
		DependencyIndexes: file_pkg_proto_noderecord_consensus_proto_depIdxs,
		MessageInfos:      file_pkg_proto_noderecord_consensus_proto_msgTypes,
	}.Build()
	File_pkg_proto_noderecord_consensus_proto = out.File
	file_pkg_proto_noderecord_consensus_proto_rawDesc = nil
	file_pkg_proto_noderecord_consensus_proto_goTypes = nil
	file_pkg_proto_noderecord_consensus_proto_depIdxs = nil
}
