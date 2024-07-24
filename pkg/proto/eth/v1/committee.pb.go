// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.2
// source: pkg/proto/eth/v1/committee.proto

// Note: largely inspired by https://github.com/prysmaticlabs/prysm/tree/develop/proto/eth/v1

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
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

// Committee is a set of validators that are assigned to a committee for a given slot.
type Committee struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Index      *wrapperspb.UInt64Value   `protobuf:"bytes,1,opt,name=index,proto3" json:"index,omitempty"`
	Slot       *wrapperspb.UInt64Value   `protobuf:"bytes,2,opt,name=slot,proto3" json:"slot,omitempty"`
	Validators []*wrapperspb.UInt64Value `protobuf:"bytes,3,rep,name=validators,proto3" json:"validators,omitempty"`
}

func (x *Committee) Reset() {
	*x = Committee{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_proto_eth_v1_committee_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Committee) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Committee) ProtoMessage() {}

func (x *Committee) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_eth_v1_committee_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Committee.ProtoReflect.Descriptor instead.
func (*Committee) Descriptor() ([]byte, []int) {
	return file_pkg_proto_eth_v1_committee_proto_rawDescGZIP(), []int{0}
}

func (x *Committee) GetIndex() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Index
	}
	return nil
}

func (x *Committee) GetSlot() *wrapperspb.UInt64Value {
	if x != nil {
		return x.Slot
	}
	return nil
}

func (x *Committee) GetValidators() []*wrapperspb.UInt64Value {
	if x != nil {
		return x.Validators
	}
	return nil
}

var File_pkg_proto_eth_v1_committee_proto protoreflect.FileDescriptor

var file_pkg_proto_eth_v1_committee_proto_rawDesc = []byte{
	0x0a, 0x20, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x74, 0x68, 0x2f,
	0x76, 0x31, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0b, 0x78, 0x61, 0x74, 0x75, 0x2e, 0x65, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x1a,
	0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xaf, 0x01, 0x0a, 0x09, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x65, 0x12, 0x32, 0x0a,
	0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55,
	0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x69, 0x6e, 0x64, 0x65,
	0x78, 0x12, 0x30, 0x0a, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x04, 0x73,
	0x6c, 0x6f, 0x74, 0x12, 0x3c, 0x0a, 0x0a, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72,
	0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0a, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72,
	0x73, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x65, 0x74, 0x68, 0x70, 0x61, 0x6e, 0x64, 0x61, 0x6f, 0x70, 0x73, 0x2f, 0x78, 0x61, 0x74, 0x75,
	0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x74, 0x68, 0x2f, 0x76,
	0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_proto_eth_v1_committee_proto_rawDescOnce sync.Once
	file_pkg_proto_eth_v1_committee_proto_rawDescData = file_pkg_proto_eth_v1_committee_proto_rawDesc
)

func file_pkg_proto_eth_v1_committee_proto_rawDescGZIP() []byte {
	file_pkg_proto_eth_v1_committee_proto_rawDescOnce.Do(func() {
		file_pkg_proto_eth_v1_committee_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_proto_eth_v1_committee_proto_rawDescData)
	})
	return file_pkg_proto_eth_v1_committee_proto_rawDescData
}

var file_pkg_proto_eth_v1_committee_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_pkg_proto_eth_v1_committee_proto_goTypes = []interface{}{
	(*Committee)(nil),              // 0: xatu.eth.v1.Committee
	(*wrapperspb.UInt64Value)(nil), // 1: google.protobuf.UInt64Value
}
var file_pkg_proto_eth_v1_committee_proto_depIdxs = []int32{
	1, // 0: xatu.eth.v1.Committee.index:type_name -> google.protobuf.UInt64Value
	1, // 1: xatu.eth.v1.Committee.slot:type_name -> google.protobuf.UInt64Value
	1, // 2: xatu.eth.v1.Committee.validators:type_name -> google.protobuf.UInt64Value
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_pkg_proto_eth_v1_committee_proto_init() }
func file_pkg_proto_eth_v1_committee_proto_init() {
	if File_pkg_proto_eth_v1_committee_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_proto_eth_v1_committee_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Committee); i {
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
			RawDescriptor: file_pkg_proto_eth_v1_committee_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_proto_eth_v1_committee_proto_goTypes,
		DependencyIndexes: file_pkg_proto_eth_v1_committee_proto_depIdxs,
		MessageInfos:      file_pkg_proto_eth_v1_committee_proto_msgTypes,
	}.Build()
	File_pkg_proto_eth_v1_committee_proto = out.File
	file_pkg_proto_eth_v1_committee_proto_rawDesc = nil
	file_pkg_proto_eth_v1_committee_proto_goTypes = nil
	file_pkg_proto_eth_v1_committee_proto_depIdxs = nil
}
