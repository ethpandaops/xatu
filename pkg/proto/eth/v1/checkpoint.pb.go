// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: pkg/proto/eth/v1/checkpoint.proto

// Note: largely inspired by
// https://github.com/prysmaticlabs/prysm/tree/develop/proto/eth/v1

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	_ "google.golang.org/protobuf/types/descriptorpb"
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

type Checkpoint struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Epoch uint64 `protobuf:"varint,1,opt,name=epoch,proto3" json:"epoch,omitempty"`
	Root  string `protobuf:"bytes,2,opt,name=root,proto3" json:"root,omitempty"`
}

func (x *Checkpoint) Reset() {
	*x = Checkpoint{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_checkpoint_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Checkpoint) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Checkpoint) ProtoMessage() {}

func (x *Checkpoint) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_checkpoint_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Checkpoint.ProtoReflect.Descriptor instead.
func (*Checkpoint) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_checkpoint_proto_rawDescGZIP(), []int{0}
}

func (x *Checkpoint) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *Checkpoint) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

type CheckpointV2 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Epoch *wrapperspb.UInt64Value `protobuf:"bytes,1,opt,name=epoch,proto3" json:"epoch,omitempty"`
	Root  string                  `protobuf:"bytes,2,opt,name=root,proto3" json:"root,omitempty"`
}

func (x *CheckpointV2) Reset() {
	*x = CheckpointV2{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_checkpoint_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CheckpointV2) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CheckpointV2) ProtoMessage() {}

func (x *CheckpointV2) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_checkpoint_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CheckpointV2.ProtoReflect.Descriptor instead.
func (*CheckpointV2) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_checkpoint_proto_rawDescGZIP(), []int{1}
}

func (x *CheckpointV2) GetEpoch() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Epoch
	}
	return nil
}

func (x *CheckpointV2) GetRoot() string {
	if x != nil {
		return x.Root
	}
	return ""
}

var File_pkg_proto_eth_v1_checkpoint_proto protoreflect.FileDescriptor

var file_pkg_proto_eth_v1_checkpoint_proto_rawDesc = []byte{
	0x0a, 0x21, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x74, 0x68, 0x2f,
	0x76, 0x31, 0x2f, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x65, 0x74, 0x68, 0x2e, 0x76, 0x31,
	0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x36, 0x0a, 0x0a, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74,
	0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x22, 0x56, 0x0a, 0x0c, 0x43, 0x68,
	0x65, 0x63, 0x6b, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x56, 0x32, 0x12, 0x32, 0x0a, 0x05, 0x65, 0x70,
	0x6f, 0x63, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74,
	0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x12,
	0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x72, 0x6f,
	0x6f, 0x74, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x65, 0x74, 0x68, 0x70, 0x61, 0x6e, 0x64, 0x61, 0x6f, 0x70, 0x73, 0x2f, 0x78, 0x61, 0x74,
	0x75, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x74, 0x68, 0x2f,
	0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_proto_eth_v1_checkpoint_proto_rawDescOnce sync.Once
	file_pkg_proto_eth_v1_checkpoint_proto_rawDescData = file_pkg_proto_eth_v1_checkpoint_proto_rawDesc
)

func file_pkg_proto_eth_v1_checkpoint_proto_rawDescGZIP() []byte {
	file_pkg_proto_eth_v1_checkpoint_proto_rawDescOnce.Do(func() {
		file_pkg_proto_eth_v1_checkpoint_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_proto_eth_v1_checkpoint_proto_rawDescData)
	})
	return file_pkg_proto_eth_v1_checkpoint_proto_rawDescData
}

var file_pkg_proto_eth_v1_checkpoint_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_pkg_proto_eth_v1_checkpoint_proto_goTypes = []any{
	(*Checkpoint)(nil),             // 0: xatu.eth.v1.Checkpoint
	(*CheckpointV2)(nil),           // 1: xatu.eth.v1.CheckpointV2
	(*wrapperspb.UInt64Value)(nil), // 2: google.protobuf.UInt64Value
}
var file_pkg_proto_eth_v1_checkpoint_proto_depIdxs = []int32{
	2, // 0: xatu.eth.v1.CheckpointV2.epoch:type_name -> google.protobuf.UInt64Value
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_pkg_proto_eth_v1_checkpoint_proto_init() }
func file_pkg_proto_eth_v1_checkpoint_proto_init() {
	if File_pkg_proto_eth_v1_checkpoint_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_proto_eth_v1_checkpoint_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*Checkpoint); i {
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
		file_pkg_proto_eth_v1_checkpoint_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*CheckpointV2); i {
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
			RawDescriptor: file_pkg_proto_eth_v1_checkpoint_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_proto_eth_v1_checkpoint_proto_goTypes,
		DependencyIndexes: file_pkg_proto_eth_v1_checkpoint_proto_depIdxs,
		MessageInfos:      file_pkg_proto_eth_v1_checkpoint_proto_msgTypes,
	}.Build()
	File_pkg_proto_eth_v1_checkpoint_proto = out.File
	file_pkg_proto_eth_v1_checkpoint_proto_rawDesc = nil
	file_pkg_proto_eth_v1_checkpoint_proto_goTypes = nil
	file_pkg_proto_eth_v1_checkpoint_proto_depIdxs = nil
}
